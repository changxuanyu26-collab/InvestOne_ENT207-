package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemberRecordSavingChoice(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := newCloudServer(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	type client struct {
		cookies map[string]*http.Cookie
		page    string
	}
	newClient := func() *client { return &client{cookies: map[string]*http.Cookie{}, page: id()} }
	request := func(cl *client, method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "https://investone.example"+path, strings.NewReader(body))
		r.Header.Set("X-InvestOne", "1")
		r.Header.Set("X-InvestOne-Page", cl.page)
		r.Header.Set("Content-Type", "application/json")
		for _, cookie := range cl.cookies {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		c.handler().ServeHTTP(w, r)
		for _, cookie := range w.Result().Cookies() {
			if cookie.MaxAge < 0 {
				delete(cl.cookies, cookie.Name)
			} else {
				cl.cookies[cookie.Name] = cookie
			}
		}
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		return w
	}
	read := func(cl *client) (State, bool) {
		w := request(cl, "GET", "/api/state", "")
		var data struct {
			State       State `json:"state"`
			SaveRecords bool  `json:"save_records"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data.State, data.SaveRecords
	}
	has := func(s State, symbol string) bool {
		for _, v := range s.Watchlist {
			if v == symbol {
				return true
			}
		}
		return false
	}
	saved := newClient()
	request(saved, "POST", "/api/account/register", `{"username":"RecordSaving","password":"test-password-123","save_records":true}`)
	read(saved)
	request(saved, "POST", "/api/watchlist", `{"symbol":"AMD","remove":false}`)
	temporary := newClient()
	request(temporary, "POST", "/api/account/login", `{"username":"RecordSaving","password":"test-password-123","save_records":false}`)
	state, enabled := read(temporary)
	if enabled || !has(state, "AMD") {
		t.Fatal("Temporary sign-in did not start from last saved data")
	}
	request(temporary, "POST", "/api/watchlist", `{"symbol":"TSLA","remove":false}`)
	state, _ = read(saved)
	if has(state, "TSLA") {
		t.Fatal("Temporary changes leaked into saved account data")
	}
	temporary.page = id()
	state, enabled = read(temporary)
	if enabled || has(state, "TSLA") || !has(state, "AMD") {
		t.Fatal("Refresh did not discard temporary changes")
	}
	request(temporary, "POST", "/api/watchlist", `{"symbol":"TSLA","remove":false}`)
	request(temporary, "POST", "/api/account/preferences", `{"save_records":true}`)
	state, enabled = read(temporary)
	if !enabled || !has(state, "TSLA") {
		t.Fatal("Enabling saving failed to save current workspace")
	}
	state, _ = read(saved)
	if !has(state, "TSLA") {
		t.Fatal("Saved changes were not visible on another device")
	}
	request(temporary, "POST", "/api/account/preferences", `{"save_records":false}`)
	read(temporary)
	request(temporary, "POST", "/api/watchlist", `{"symbol":"META","remove":false}`)
	state, _ = read(temporary)
	if !has(state, "META") {
		t.Fatal("Pausing saving did not retain current-page changes")
	}
	temporary.page = id()
	state, enabled = read(temporary)
	if enabled || has(state, "META") || !has(state, "TSLA") {
		t.Fatal("Paused saving modified persistent account records")
	}
	cancel()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	c, err = newCloudServer(ctx2, root)
	if err != nil {
		t.Fatal(err)
	}
	state, _ = read(saved)
	if !has(state, "TSLA") || has(state, "META") {
		t.Fatal("Restart did not preserve only saved records")
	}
}
