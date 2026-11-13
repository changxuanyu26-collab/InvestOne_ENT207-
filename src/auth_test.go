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

func nameLogin(t *testing.T, c *cloudServer, name string, cookies ...*http.Cookie) []*http.Cookie {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"username": name})
	w := cloudRequest(c.handler(), "POST", "/api/login", string(b), cookies...)
	if w.Code != 200 {
		t.Fatalf("Name entry failed: %d %s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}
func namedState(t *testing.T, c *cloudServer, cookies ...*http.Cookie) (State, string) {
	t.Helper()
	w := cloudRequest(c.handler(), "GET", "/api/state", "", cookies...)
	if w.Code != 200 {
		t.Fatalf("State failed: %d", w.Code)
	}
	var d struct {
		State         State  `json:"state"`
		Username      string `json:"username"`
		SignedIn      bool   `json:"signed_in"`
		Authenticated bool   `json:"authenticated"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if !d.SignedIn || d.Authenticated {
		t.Fatal("Name entry must be identified without claiming password authentication")
	}
	return d.State, d.Username
}
func TestNameEntryIsolationReturnAndRestart(t *testing.T) {
	t.Setenv("INVESTONE_CLOUD", "1")
	t.Setenv("INVESTONE_SHARED_AI_KEY", "fixture-course-key")
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := newCloudServer(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	a := nameLogin(t, c, "  张同学 Sam  ")
	state, name := namedState(t, c, a...)
	if name != "张同学 Sam" {
		t.Fatal("Name was not trimmed or Chinese names were rejected")
	}
	var browser *http.Cookie
	for _, cookie := range a {
		if cookie.Name == cloudCookie {
			browser = cookie
		}
		if cookie.Name == namedAccessCookie && (!cookie.Secure || !cookie.HttpOnly) {
			t.Fatal("Workspace cookie is not protected")
		}
	}
	s, err := c.load(browser.Value)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.app.store.update(func(v *State) error { v.Profile.Interests = "BROWSER_A_FIXTURE"; return nil }); err != nil {
		t.Fatal(err)
	}
	// Matching names are display labels, never global keys to another person's data.
	b := nameLogin(t, c, "张同学 Sam")
	other, _ := namedState(t, c, b...)
	if other.Profile.Interests == "BROWSER_A_FIXTURE" {
		t.Fatal("Same-name visitors shared private data")
	}
	logout := cloudRequest(c.handler(), "POST", "/api/logout", "{}", a...)
	if logout.Code != 200 || pickCookie(t, logout, namedAccessCookie).MaxAge != -1 {
		t.Fatal("Sign-out failed")
	}
	returned := nameLogin(t, c, "张同学 Sam", browser)
	saved, _ := namedState(t, c, returned...)
	if saved.Profile.Interests != "BROWSER_A_FIXTURE" {
		t.Fatal("Returning browser lost work")
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	restarted, err := newCloudServer(ctx2, root)
	if err != nil {
		t.Fatal(err)
	}
	persisted, _ := namedState(t, restarted, returned...)
	if persisted.Profile.Interests != saved.Profile.Interests {
		t.Fatal("Restart lost saved work")
	}
	_ = state
	if _, err := os.Stat(filepath.Join(root, "accounts.json")); !os.IsNotExist(err) {
		t.Fatal("Name entry should not create password accounts")
	}
	info, err := os.Stat(filepath.Join(root, "browser-names.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("Name records should be private")
	}
}
func TestNameEntryValidationAndExistingGuestData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := newCloudServer(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "   ", strings.Repeat("张", 61), "Sam\nChang"} {
		b, _ := json.Marshal(map[string]string{"username": name})
		if w := cloudRequest(c.handler(), "POST", "/api/login", string(b)); w.Code != 400 {
			t.Fatal("Invalid display name accepted")
		}
	}
	guest := cloudRequest(c.handler(), "GET", "/api/state", "")
	browser := pickCookie(t, guest, cloudCookie)
	s, _ := c.load(browser.Value)
	s.app.store.update(func(v *State) error { v.Profile.Interests = "EXISTING_GUEST_FIXTURE"; return nil })
	cookies := nameLogin(t, c, "A", browser)
	saved, name := namedState(t, c, cookies...)
	if name != "A" || saved.Profile.Interests != "EXISTING_GUEST_FIXTURE" {
		t.Fatal("Short name or existing guest data was not preserved")
	}
	if w := cloudRequest(c.handler(), "POST", "/api/register", `{}`); w.Code != 410 {
		t.Fatal("Old registration should be retired")
	}
}
func TestNameEntryPreservesExistingSignedInWorkspace(t *testing.T) {
	root := t.TempDir()
	token := id()
	workspace := id()
	legacy := map[string]any{"users": map[string]any{"olduser": map[string]string{"username": "OldUser", "workspace": workspace, "password_hash": "unused-legacy-hash"}}, "sessions": map[string]accountSession{sessionDigest(token): {User: "olduser", Expires: time.Now().Add(time.Hour)}}}
	raw, _ := json.Marshal(legacy)
	if err := os.WriteFile(filepath.Join(root, "accounts.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := openStore(filepath.Join(root, "account-workspaces", workspace+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = store.update(func(v *State) error { v.Profile.Interests = "LEGACY_WORKSPACE_FIXTURE"; return nil }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := newCloudServer(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	migration := cloudRequest(c.handler(), "GET", "/api/auth/session", "", &http.Cookie{Name: accountCookie, Value: token})
	if migration.Code != 200 || !strings.Contains(migration.Body.String(), `"signed_in":true`) {
		t.Fatal("Existing signed-in browser was not restored")
	}
	cookies := migration.Result().Cookies()
	saved, name := namedState(t, c, cookies...)
	if name != "OldUser" || saved.Profile.Interests != "LEGACY_WORKSPACE_FIXTURE" {
		t.Fatal("Legacy data not retained")
	}
	stranger := nameLogin(t, c, "OldUser")
	different, _ := namedState(t, c, stranger...)
	if different.Profile.Interests == saved.Profile.Interests {
		t.Fatal("Typing a legacy name unlocked someone else's data")
	}
	after, _ := os.ReadFile(filepath.Join(root, "accounts.json"))
	if string(after) != string(raw) {
		t.Fatal("Legacy data should remain untouched")
	}
}
