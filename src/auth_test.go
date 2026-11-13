package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testAccountPassword = "account-password-fixture-2026"

func accountRequest(t *testing.T, c *cloudServer, path, user, password string, cookies ...*http.Cookie) *http.Cookie {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"username": user, "password": password})
	w := cloudRequest(c.handler(), "POST", path, string(b), cookies...)
	if w.Code != 200 {
		t.Fatalf("Account operation %s failed: %d %s", path, w.Code, w.Body.String())
	}
	return pickCookie(t, w, accountCookie)
}
func accountState(t *testing.T, c *cloudServer, cookie *http.Cookie) (State, []*http.Cookie) {
	t.Helper()
	w := cloudRequest(c.handler(), "GET", "/api/state", "", cookie)
	if w.Code != 200 {
		t.Fatalf("State returned %d", w.Code)
	}
	var data struct {
		State         State  `json:"state"`
		Username      string `json:"username"`
		Authenticated bool   `json:"authenticated"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Authenticated || data.Username == "Guest" {
		t.Fatal("Account identity missing")
	}
	return data.State, w.Result().Cookies()
}
func TestAccountsLoginIsolationLogoutAndRestart(t *testing.T) {
	t.Setenv("INVESTONE_CLOUD", "1")
	t.Setenv("INVESTONE_SHARED_AI_KEY", "fixture-course-key")
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := newCloudServer(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	access := cloudRequest(c.handler(), "GET", "/api/auth/session", "")
	if !strings.Contains(access.Body.String(), `"authenticated":false`) || !strings.Contains(access.Body.String(), `"guest":false`) {
		t.Fatal("Fresh visitor should get an explicit entry choice")
	}
	guest := cloudRequest(c.handler(), "GET", "/api/state", "")
	guestID := pickCookie(t, guest, cloudCookie)
	guestAuth := pickCookie(t, guest, "investone_session")
	profile := `{"risk":"Low","horizon":"10 years","knowledge":"Beginner","interests":"GUEST_COPY_FIXTURE","email":"","model":"DeepSeek-V4-Flash","max_tokens":240}`
	if w := cloudRequest(c.handler(), "POST", "/api/profile", profile, guestID, guestAuth); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	a := accountRequest(t, c, "/api/register", "ResearcherA", testAccountPassword, guestID)
	if !a.HttpOnly || !a.Secure || a.MaxAge != 7*86400 {
		t.Fatal("Account cookie protection missing")
	}
	state, cookies := accountState(t, c, a)
	if state.Profile.Interests != "GUEST_COPY_FIXTURE" {
		t.Fatal("Guest workspace not copied")
	}
	b := accountRequest(t, c, "/api/register", "ResearcherB", testAccountPassword)
	stateB, _ := accountState(t, c, b)
	if stateB.Profile.Interests == state.Profile.Interests {
		t.Fatal("Account data leaked")
	}
	u, _ := c.accounts.resolve(a.Value)
	forgedGuest := cloudRequest(c.handler(), "GET", "/api/state", "", &http.Cookie{Name: cloudCookie, Value: u.Workspace})
	if strings.Contains(forgedGuest.Body.String(), "GUEST_COPY_FIXTURE") {
		t.Fatal("Guest cookie reached account storage")
	}
	otherDevice := accountRequest(t, c, "/api/login", "researchera", testAccountPassword)
	resumed, _ := accountState(t, c, otherDevice)
	if resumed.Profile.Interests != state.Profile.Interests {
		t.Fatal("Cross-device sign-in lost workspace")
	}
	wrong := cloudRequest(c.handler(), "POST", "/api/login", `{"username":"ResearcherA","password":"incorrect-password-value"}`)
	if wrong.Code != 401 {
		t.Fatal("Incorrect password accepted")
	}
	bad := cloudRequest(c.handler(), "POST", "/api/login", `{"username":"ResearcherA"}`)
	if bad.Code != 400 {
		t.Fatal("Password-free legacy cloud login accepted")
	}
	logoutCookies := append([]*http.Cookie{a}, cookies...)
	logout := cloudRequest(c.handler(), "POST", "/api/logout", "{}", logoutCookies...)
	if logout.Code != 200 {
		t.Fatal("Logout failed")
	}
	if w := cloudRequest(c.handler(), "GET", "/api/state", "", a); w.Code != 401 {
		t.Fatal("Logged-out credential still works")
	}
	accountState(t, c, otherDevice) // Other devices remain signed in.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	restarted, err := newCloudServer(ctx2, root)
	if err != nil {
		t.Fatal(err)
	}
	afterRestart, _ := accountState(t, restarted, otherDevice)
	if afterRestart.Profile.Interests != state.Profile.Interests {
		t.Fatal("Workspace/session did not survive restart")
	}
	stored, err := os.ReadFile(filepath.Join(root, "accounts.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), testAccountPassword) || strings.Contains(string(stored), otherDevice.Value) || strings.Contains(string(stored), "fixture-course-key") {
		t.Fatal("Plaintext secret persisted")
	}
	info, _ := os.Stat(filepath.Join(root, "accounts.json"))
	if info.Mode().Perm() != 0600 {
		t.Fatal("Account file must be private")
	}
	for _, p := range []string{"/api/state", "/api/export"} {
		w := cloudRequest(restarted.handler(), "GET", p, "", otherDevice)
		if p == "/api/export" {
			s := cloudRequest(restarted.handler(), "GET", "/api/state", "", otherDevice)
			w = cloudRequest(restarted.handler(), "GET", p, "", otherDevice, pickCookie(t, s, "investone_session"))
		}
		if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "fixture-course-key") {
			t.Fatal("Credentials exposed through application API")
		}
	}
}

func TestAccountValidationExpiryAndGuestReturn(t *testing.T) {
	t.Setenv("INVESTONE_CLOUD", "1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := newCloudServer(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := c.handler()
	for _, body := range []string{`{"username":"../escape","password":"account-password-fixture"}`, `{"username":"ValidUser","password":"short"}`, `{"username":"ValidUser","password":"` + strings.Repeat("a", 73) + `"}`} {
		if w := cloudRequest(h, "POST", "/api/register", body); w.Code != 400 {
			t.Fatal("Invalid credentials accepted")
		}
	}
	cookie := accountRequest(t, c, "/api/register", "ValidUser", testAccountPassword)
	if w := cloudRequest(h, "POST", "/api/register", `{"username":"validuser","password":"account-password-fixture"}`); w.Code != 409 {
		t.Fatal("Duplicate case-insensitive username accepted")
	}
	c.accounts.mu.Lock()
	key := sessionDigest(cookie.Value)
	s := c.accounts.data.Sessions[key]
	s.Expires = time.Now().Add(-time.Second)
	c.accounts.data.Sessions[key] = s
	c.accounts.mu.Unlock()
	if w := cloudRequest(h, "GET", "/api/state", "", cookie); w.Code != 401 {
		t.Fatal("Expired session accepted or silently became a guest")
	}
	guest := cloudRequest(h, "POST", "/api/guest", "{}")
	g := pickCookie(t, guest, guestAccessCookie)
	if w := cloudRequest(h, "GET", "/api/auth/session", "", g); !strings.Contains(w.Body.String(), `"guest":true`) {
		t.Fatal("Explicit guest entry not remembered")
	}
	logout := cloudRequest(h, "POST", "/api/logout", "{}", g)
	if pickCookie(t, logout, guestAccessCookie).MaxAge != -1 {
		t.Fatal("Sign-out should return to entry choice after reload")
	}
	for i := 0; i < 31; i++ {
		cloudRequest(h, "POST", "/api/login", `{}`)
	}
	if w := cloudRequest(h, "POST", "/api/login", `{}`); w.Code != 429 {
		t.Fatal("Login throttling absent")
	}
}
