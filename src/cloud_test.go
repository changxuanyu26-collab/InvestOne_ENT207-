package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudRequest(handler http.Handler, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://investone.example"+path, strings.NewReader(body))
	req.Header.Set("X-InvestOne", "1")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://investone.example")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}
func pickCookie(t *testing.T, w *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("Cookie %s missing", name)
	return nil
}
func TestCloudLandingAndSessionIsolation(t *testing.T) {
	t.Setenv("INVESTONE_CLOUD", "1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cloud, e := newCloudServer(ctx, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	h := cloud.handler()
	landing := cloudRequest(h, "GET", "/", "")
	if landing.Code != 200 || !strings.Contains(landing.Body.String(), "Try Our App") {
		t.Fatalf("Landing %d: %s", landing.Code, landing.Body.String())
	}
	appPage := cloudRequest(h, "GET", "/app/", "")
	if appPage.Code != 200 || !strings.Contains(appPage.Body.String(), "/app.js") {
		t.Fatalf("App page %d", appPage.Code)
	}
	// Guests can see real provider-backed research without a login.
	guest := cloudRequest(h, "GET", "/api/state", "")
	if guest.Code != 200 {
		t.Fatalf("Guest access: %d %s", guest.Code, guest.Body.String())
	}
	if !pickCookie(t, guest, "investone_session").Secure {
		t.Fatal("Guest auth cookie must be secure")
	}
	loginPayload := `{"username":"RedBeanCoix","key":"key-visitor-a"}`
	visitorA := cloudRequest(h, "POST", "/api/login", loginPayload)
	if visitorA.Code != 200 {
		t.Fatalf("A login: %d %s", visitorA.Code, visitorA.Body.String())
	}
	cookieA := pickCookie(t, visitorA, cloudCookie)
	authA := pickCookie(t, visitorA, "investone_session")
	if !cookieA.Secure || !authA.Secure || !cookieA.HttpOnly || !authA.HttpOnly {
		t.Fatal("Cloud cookies must be secure and HTTP-only")
	}
	visitorB := cloudRequest(h, "POST", "/api/login", `{"username":"RedBeanCoix","key":"key-visitor-b"}`)
	if visitorB.Code != 200 {
		t.Fatalf("B login: %d %s", visitorB.Code, visitorB.Body.String())
	}
	cookieB := pickCookie(t, visitorB, cloudCookie)
	authB := pickCookie(t, visitorB, "investone_session")
	if cookieB.Value == cookieA.Value {
		t.Fatal("Visitors share a browser session")
	}
	payload := `{"risk":"Conservative","horizon":"5+ years","knowledge":"Intermediate","interests":"Technology","email":"","model":"DeepSeek-V4-Flash","max_tokens":800}`
	update := cloudRequest(h, "POST", "/api/profile", payload, cookieA, authA)
	if update.Code != 200 {
		t.Fatalf("Profile update: %d %s", update.Code, update.Body.String())
	}
	stateA := cloudRequest(h, "GET", "/api/state", "", cookieA, authA)
	stateB := cloudRequest(h, "GET", "/api/state", "", cookieB, authB)
	var a, b struct {
		State State `json:"state"`
	}
	if e = json.Unmarshal(stateA.Body.Bytes(), &a); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(stateB.Body.Bytes(), &b); e != nil {
		t.Fatal(e)
	}
	if a.State.Profile.Risk != "Conservative" || b.State.Profile.Risk == "Conservative" {
		t.Fatal("Browser state leakage or failed save")
	}
	if cloud.sessions[cookieA.Value].app.getKey() == cloud.sessions[cookieB.Value].app.getKey() {
		t.Fatal("Browser key leak")
	}
	malicious := httptest.NewRequest("POST", "https://investone.example/api/login", strings.NewReader(loginPayload))
	malicious.Header.Set("X-InvestOne", "1")
	malicious.Header.Set("Origin", "https://evil.example")
	reject := httptest.NewRecorder()
	h.ServeHTTP(reject, malicious)
	if reject.Code != 403 {
		t.Fatalf("CSRF origin not blocked: %d", reject.Code)
	}
}

func TestCloudCrossSiteTopLevelNavigation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cloud, err := newCloudServer(ctx, t.TempDir())
	if err != nil { t.Fatal(err) }
	handler := cloud.handler()
	request := func(method,path,mode,dest string) int {
		r:=httptest.NewRequest(method,"https://investone.example"+path,nil)
		r.Header.Set("Sec-Fetch-Site","cross-site")
		r.Header.Set("Sec-Fetch-Mode",mode)
		r.Header.Set("Sec-Fetch-Dest",dest)
		r.Header.Set("X-InvestOne","1")
		w:=httptest.NewRecorder()
		handler.ServeHTTP(w,r)
		return w.Code
	}
	for _,path:= range []string{"/","/app/","/app","/landing"} {
		if got:=request("GET",path,"navigate","document"); got!=200 {
			t.Errorf("Cross-site top-level navigation %s should succeed; got %d",path,got)
		}
	}
	for _,tc:=range []struct{method,path,mode,dest string}{
		{"GET","/api/state","cors","empty"},
		{"POST","/api/login","navigate","document"},
		{"GET","/app.js","no-cors","script"},
		{"GET","/api/state","navigate","document"},
		{"GET","/app/","no-cors","iframe"},
	} {
		if got:=request(tc.method,tc.path,tc.mode,tc.dest);got!=http.StatusForbidden{
			t.Errorf("Unsafe cross-site request %s %s (%s/%s) returned %d",tc.method,tc.path,tc.mode,tc.dest,got)
		}
	}
}
