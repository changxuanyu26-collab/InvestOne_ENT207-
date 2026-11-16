package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestMemberIsolationPersistenceAndRevocation(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, e := newCloudServer(ctx, root)
	if e != nil {
		t.Fatal(e)
	}
	auth := func(server *cloudServer, path, name, password string) []*http.Cookie {
		t.Helper()
		b, _ := json.Marshal(map[string]string{"username": name, "password": password})
		w := cloudRequest(server.handler(), "POST", path, string(b))
		if w.Code != 200 {
			t.Fatalf("auth %d %s", w.Code, w.Body.String())
		}
		return w.Result().Cookies()
	}
	cookies := auth(c, "/api/account/register", "Alice", "correct-password")
	w := cloudRequest(c.handler(), "GET", "/api/state", "", cookies...)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cookies = append(cookies[:1], w.Result().Cookies()...)
	config := `{"universe":"technology","range":"5y","top_n":2,"fee_bps":12,"momentum":60,"low_vol":20,"trend":20,"risk_off":true}`
	w = cloudRequest(c.handler(), "POST", "/api/quant/config", config, cookies...)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = cloudRequest(c.handler(), "POST", "/api/watchlist", `{"symbol":"AMD","remove":false}`, cookies...)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	bad := cloudRequest(c.handler(), "POST", "/api/account/login", `{"username":"Alice","password":"wrong-password"}`)
	if bad.Code != 401 {
		t.Fatal("wrong password accepted")
	}
	duplicate := cloudRequest(c.handler(), "POST", "/api/account/register", `{"username":"ALICE","password":"another-password"}`)
	if duplicate.Code != 409 {
		t.Fatal("duplicate account accepted")
	}
	cancel()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	c2, e := newCloudServer(ctx2, root)
	if e != nil {
		t.Fatal(e)
	}
	restored := auth(c2, "/api/account/login", "alice", "correct-password")
	w = cloudRequest(c2.handler(), "GET", "/api/state", "", restored...)
	var data struct {
		State         State `json:"state"`
		Authenticated bool  `json:"authenticated"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &data); e != nil {
		t.Fatal(e)
	}
	if !data.Authenticated || data.State.QuantConfig == nil || data.State.QuantConfig.TopN != 2 {
		t.Fatal("account settings not restored", w.Body.String())
	}
	found := false
	for _, s := range data.State.Watchlist {
		if s == "AMD" {
			found = true
		}
	}
	if !found {
		t.Fatal("watchlist not restored")
	}
	bob := auth(c2, "/api/account/register", "Bob", "another-password")
	w = cloudRequest(c2.handler(), "GET", "/api/state", "", bob...)
	data.State = State{}
	json.Unmarshal(w.Body.Bytes(), &data)
	if data.State.QuantConfig != nil {
		t.Fatal("account data leaked")
	}
	w = cloudRequest(c2.handler(), "POST", "/api/logout", `{}`, restored...)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = cloudRequest(c2.handler(), "GET", "/api/state", "", restored...)
	if w.Code != 401 {
		t.Fatal("revoked cookie accepted", w.Code)
	}
	// A public display name must never unlock a registered account.
	visitor := nameLogin(t, c2, "Alice")
	w = cloudRequest(c2.handler(), "GET", "/api/state", "", visitor...)
	data.State = State{}
	json.Unmarshal(w.Body.Bytes(), &data)
	if data.State.QuantConfig != nil || data.Authenticated {
		t.Fatal("name-only login unlocked account")
	}
}
