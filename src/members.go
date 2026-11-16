package main

import (
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const memberCookie = "investone_member"

type member struct {
	Name      string
	Hash      []byte
	Workspace string
}
type memberData struct {
	Users    map[string]member
	Sessions map[string]accountSession
}
type memberStore struct {
	sync.Mutex
	path     string
	data     memberData
	since    time.Time
	attempts int
}

func openMembers(root string) (*memberStore, error) {
	m := &memberStore{path: filepath.Join(root, "members.json"), data: memberData{Users: map[string]member{}, Sessions: map[string]accountSession{}}}
	b, e := os.ReadFile(m.path)
	if e == nil {
		e = json.Unmarshal(b, &m.data)
	}
	if os.IsNotExist(e) {
		e = nil
	}
	if e == nil && (m.data.Users == nil || m.data.Sessions == nil) {
		e = errors.New("Invalid account storage")
	}
	return m, e
}
func (m *memberStore) save(d memberData) error {
	b, e := json.Marshal(d)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(m.path), ".members-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e == nil {
		e = os.Rename(f.Name(), m.path)
	}
	if e == nil {
		m.data = d
	}
	return e
}
func (m *memberStore) copy() memberData {
	d := memberData{Users: map[string]member{}, Sessions: map[string]accountSession{}}
	for k, v := range m.data.Users {
		d.Users[k] = v
	}
	for k, v := range m.data.Sessions {
		if time.Now().Before(v.Expires) {
			d.Sessions[k] = v
		}
	}
	return d
}
func (c *cloudServer) memberIdentity(r *http.Request) (account, bool) {
	cookie, e := r.Cookie(memberCookie)
	if e != nil {
		return account{}, false
	}
	m := c.members
	m.Lock()
	defer m.Unlock()
	s, ok := m.data.Sessions[sessionDigest(cookie.Value)]
	if !ok || time.Now().After(s.Expires) {
		return account{}, false
	}
	u, ok := m.data.Users[s.User]
	return account{Username: u.Name, Workspace: u.Workspace, Temporary: s.Temporary, SessionID: sessionDigest(cookie.Value)}, ok
}
func (c *cloudServer) memberAuth(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	if p != "/api/account/register" && p != "/api/account/login" && p != "/api/account/preferences" && p != "/api/logout" && p != "/api/guest" {
		return false
	}
	if r.Method != "POST" {
		fail(w, 405, errors.New("Method not allowed"))
		return true
	}
	if p == "/api/account/preferences" {
		c.memberPreferences(w, r)
		return true
	}
	m := c.members
	if p == "/api/logout" || p == "/api/guest" {
		if cookie, e := r.Cookie(memberCookie); e == nil {
			m.Lock()
			d := m.copy()
			delete(d.Sessions, sessionDigest(cookie.Value))
			e = m.save(d)
			m.Unlock()
			if e != nil {
				fail(w, 500, errors.New("Unable to sign out"))
				return true
			}
		}
		setAccessCookie(w, memberCookie, "", -1)
		return false
	}
	var v struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		SaveRecords *bool  `json:"save_records"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if e := decode(w, r, &v); e != nil {
		fail(w, 400, e)
		return true
	}
	v.Username = strings.TrimSpace(v.Username)
	if !validDisplayName(v.Username) || len(v.Password) < 10 || len(v.Password) > 72 {
		fail(w, 400, errors.New("Use a name of 1–60 characters and a password of 10–72 bytes"))
		return true
	}
	m.Lock()
	defer m.Unlock()
	if time.Since(m.since) > time.Minute {
		m.since = time.Now()
		m.attempts = 0
	}
	m.attempts++
	if m.attempts > 30 {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, errors.New("Too many sign-in attempts. Please try again in a minute"))
		return true
	}
	key := strings.ToLower(v.Username)
	d := m.copy()
	u, exists := d.Users[key]
	if p == "/api/account/register" {
		if exists {
			fail(w, 409, errors.New("This username is unavailable"))
			return true
		}
		h, e := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
		if e != nil {
			fail(w, 500, errors.New("Unable to create account"))
			return true
		}
		u = member{Name: v.Username, Hash: h, Workspace: id()}
		d.Users[key] = u
	} else {
		hash := u.Hash
		if !exists {
			hash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
		}
		e := bcrypt.CompareHashAndPassword(hash, []byte(v.Password))
		if e != nil || !exists {
			fail(w, 401, errors.New("Incorrect username or password"))
			return true
		}
	}
	// Limit outstanding sessions per account without affecting other users.
	count := 0
	for _, s := range d.Sessions {
		if s.User == key {
			count++
		}
	}
	if count >= 20 {
		for k, s := range d.Sessions {
			if s.User == key {
				delete(d.Sessions, k)
			}
		}
	}
	token := id()
	d.Sessions[sessionDigest(token)] = accountSession{User: key, Expires: time.Now().Add(30 * 24 * time.Hour), Temporary: v.SaveRecords != nil && !*v.SaveRecords}
	if e := m.save(d); e != nil {
		fail(w, 500, errors.New("Unable to save account"))
		return true
	}
	setAccessCookie(w, memberCookie, token, 30*86400)
	setAccessCookie(w, "investone_session", "", -1)
	setAccessCookie(w, namedAccessCookie, "", -1)
	setAccessCookie(w, accountCookie, "", -1)
	send(w, 200, map[string]any{"ok": true, "username": u.Name})
	return true
}

// Saving can be selected for each sign-in. Temporary changes stay in server
// memory for this page, seeded from the last saved account workspace.
func (c *cloudServer) temporaryMember(n account, r *http.Request) (*cloudSession, error) {
	saved, err := c.loadAccount(n.Workspace)
	if err != nil {
		return nil, err
	}
	seed := saved.app.store.snapshot()
	page := r.Header.Get("X-InvestOne-Page")
	if !cloudTokenPattern.MatchString(page) {
		page = n.SessionID
	}
	return c.loadTransient("member-"+n.Workspace+"-"+n.SessionID+"-"+page, &seed)
}
func (c *cloudServer) memberPreferences(w http.ResponseWriter, r *http.Request) {
	n, ok := c.memberIdentity(r)
	if !ok {
		fail(w, 401, errors.New("Sign in to change record saving"))
		return
	}
	var input struct {
		SaveRecords *bool `json:"save_records"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := decode(w, r, &input); err != nil || input.SaveRecords == nil {
		fail(w, 400, errors.New("Choose whether to save records"))
		return
	}
	if *input.SaveRecords == !n.Temporary {
		send(w, 200, map[string]bool{"ok": true})
		return
	}
	current, _, err := c.workspace(w, r)
	if err != nil {
		fail(w, 500, errors.New("Workspace unavailable"))
		return
	}
	state := current.app.store.snapshot()
	state.Messages = []Message{}
	if *input.SaveRecords {
		saved, err := c.loadAccount(n.Workspace)
		if err != nil {
			fail(w, 500, errors.New("Saved workspace unavailable"))
			return
		}
		if err := saved.app.store.update(func(s *State) error { *s = clone(state); return nil }); err != nil {
			fail(w, 500, errors.New("Unable to save records"))
			return
		}
	} else {
		temporary, err := c.temporaryMember(n, r)
		if err != nil {
			fail(w, 500, errors.New("Temporary workspace unavailable"))
			return
		}
		if err := temporary.app.store.update(func(s *State) error { *s = clone(state); return nil }); err != nil {
			fail(w, 500, errors.New("Unable to pause saving"))
			return
		}
	}
	m := c.members
	m.Lock()
	defer m.Unlock()
	d := m.copy()
	session, valid := d.Sessions[n.SessionID]
	if !valid {
		fail(w, 401, errors.New("Sign in again"))
		return
	}
	session.Temporary = !*input.SaveRecords
	d.Sessions[n.SessionID] = session
	if err := m.save(d); err != nil {
		fail(w, 500, errors.New("Unable to update saving preference"))
		return
	}
	send(w, 200, map[string]bool{"ok": true, "save_records": *input.SaveRecords})
}
