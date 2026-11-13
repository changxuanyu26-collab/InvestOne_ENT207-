package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const accountCookie = "investone_account"
const guestAccessCookie = "investone_guest_access"
const accountLifetime = 7 * 24 * time.Hour

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$`)
var errExpiredAccount = errors.New("Your session has expired. Please sign in again.")

type account struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Workspace    string `json:"workspace"`
}
type accountSession struct {
	User    string    `json:"user"`
	Expires time.Time `json:"expires"`
}
type accountData struct {
	Users    map[string]account        `json:"users"`
	Sessions map[string]accountSession `json:"sessions"`
}
type authAttempt struct {
	Start time.Time
	Count int
}
type accountStore struct {
	mu       sync.Mutex
	path     string
	data     accountData
	attempts map[string]authAttempt
	gate     chan struct{}
}

func openAccounts(root string) (*accountStore, error) {
	a := &accountStore{path: filepath.Join(root, "accounts.json"), data: accountData{Users: map[string]account{}, Sessions: map[string]accountSession{}}, attempts: map[string]authAttempt{}, gate: make(chan struct{}, 2)}
	b, err := os.ReadFile(a.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if json.Unmarshal(b, &a.data) != nil || a.data.Users == nil || a.data.Sessions == nil {
			return nil, errors.New("Invalid account storage")
		}
		for name, user := range a.data.Users {
			if !usernamePattern.MatchString(user.Username) || name != strings.ToLower(user.Username) || !cloudTokenPattern.MatchString(user.Workspace) {
				return nil, errors.New("Invalid account storage")
			}
			if _, err := bcrypt.Cost([]byte(user.PasswordHash)); err != nil {
				return nil, errors.New("Invalid account storage")
			}
		}
	}
	return a, nil
}
func sessionDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (a *accountStore) nextLocked() accountData {
	n := accountData{Users: map[string]account{}, Sessions: map[string]accountSession{}}
	for k, v := range a.data.Users {
		n.Users[k] = v
	}
	for k, v := range a.data.Sessions {
		if time.Now().Before(v.Expires) {
			n.Sessions[k] = v
		}
	}
	return n
}

// Persist hashes and session digests atomically; plaintext credentials never go to disk.
func (a *accountStore) saveLocked(n accountData) error {
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(a.path), ".accounts-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), a.path); err != nil {
		return err
	}
	a.data = n
	return nil
}
func (a *accountStore) resolve(token string) (account, bool) {
	if !cloudTokenPattern.MatchString(token) {
		return account{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.data.Sessions[sessionDigest(token)]
	if !ok || !time.Now().Before(s.Expires) {
		return account{}, false
	}
	u, ok := a.data.Users[s.User]
	return u, ok
}
func (a *accountStore) revoke(token string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := a.nextLocked()
	delete(n.Sessions, sessionDigest(token))
	return a.saveLocked(n)
}
func (a *accountStore) allow(r *http.Request) bool {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	// Railway appends the real connecting IP to X-Forwarded-For.
	if forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ","); len(forwarded) > 0 {
		if last := strings.TrimSpace(forwarded[len(forwarded)-1]); net.ParseIP(last) != nil {
			ip = last
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, v := range a.attempts {
		if now.Sub(v.Start) > 15*time.Minute {
			delete(a.attempts, k)
		}
	}
	v := a.attempts[ip]
	if v.Start.IsZero() {
		v.Start = now
	}
	v.Count++
	a.attempts[ip] = v
	return v.Count <= 30 && len(a.attempts) <= 10000
}
func setAccountCookie(w http.ResponseWriter, token string, age int) {
	http.SetCookie(w, &http.Cookie{Name: accountCookie, Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func clearGuestAccess(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: guestAccessCookie, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
func (c *cloudServer) identity(r *http.Request) (account, bool) {
	if cookie, err := r.Cookie(accountCookie); err == nil {
		return c.accounts.resolve(cookie.Value)
	}
	return account{}, false
}
func (c *cloudServer) workspace(w http.ResponseWriter, r *http.Request) (*cloudSession, *account, error) {
	if cookie, err := r.Cookie(accountCookie); err == nil && cookie.Value != "" {
		u, ok := c.accounts.resolve(cookie.Value)
		if !ok {
			return nil, nil, errExpiredAccount
		}
		s, err := c.loadAccount(u.Workspace)
		return s, &u, err
	}
	s, err := c.browserSession(w, r)
	return s, nil, err
}
func (c *cloudServer) auth(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path != "/api/auth/session" && path != "/api/register" && path != "/api/login" && path != "/api/logout" && path != "/api/guest" {
		return false
	}
	if path == "/api/auth/session" {
		if r.Method != "GET" {
			fail(w, 405, errors.New("Method not allowed"))
			return true
		}
		u, ok := c.identity(r)
		g, err := r.Cookie(guestAccessCookie)
		if cookie, e := r.Cookie(accountCookie); e == nil && cookie.Value != "" && !ok {
			setAccountCookie(w, "", -1)
		}
		send(w, 200, map[string]any{"cloud": true, "authenticated": ok, "username": u.Username, "guest": !ok && err == nil && g.Value == "1"})
		return true
	}
	if r.Method != "POST" {
		fail(w, 405, errors.New("Method not allowed"))
		return true
	}
	if path == "/api/logout" || path == "/api/guest" {
		if cookie, err := r.Cookie(accountCookie); err == nil && cookie.Value != "" {
			if err := c.accounts.revoke(cookie.Value); err != nil {
				fail(w, 500, errors.New("Unable to sign out. Please try again."))
				return true
			}
		}
		// Revoke only this browser's legacy/guest session, never other devices.
		if cookie, err := r.Cookie("investone_session"); err == nil {
			if s, _, e := c.workspace(w, r); e == nil {
				s.app.mu.Lock()
				delete(s.app.sessions, cookie.Value)
				s.app.mu.Unlock()
			}
		}
		setAccountCookie(w, "", -1)
		if path == "/api/logout" {
			clearGuestAccess(w)
		}
		http.SetCookie(w, &http.Cookie{Name: "investone_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
		if path == "/api/guest" {
			http.SetCookie(w, &http.Cookie{Name: guestAccessCookie, Value: "1", Path: "/", MaxAge: 30 * 86400, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
		}
		send(w, 200, map[string]bool{"ok": true})
		return true
	}
	if !c.accounts.allow(r) {
		w.Header().Set("Retry-After", "900")
		fail(w, 429, errors.New("Too many sign-in attempts. Please try again in 15 minutes."))
		return true
	}
	select {
	case c.accounts.gate <- struct{}{}:
		defer func() { <-c.accounts.gate }()
	default:
		w.Header().Set("Retry-After", "3")
		fail(w, 429, errors.New("Sign-in is busy. Please try again shortly."))
		return true
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := decode(w, r, &input); err != nil {
		fail(w, 400, err)
		return true
	}
	input.Username = strings.TrimSpace(input.Username)
	if !usernamePattern.MatchString(input.Username) {
		fail(w, 400, errors.New("Use 3–32 letters, numbers, underscores or hyphens, starting with a letter or number."))
		return true
	}
	if len(input.Password) < 12 || len(input.Password) > 72 {
		fail(w, 400, errors.New("Use a password of 12–72 bytes (12–72 characters for English text)."))
		return true
	}
	name := strings.ToLower(input.Username)
	c.accounts.mu.Lock()
	u, exists := c.accounts.data.Users[name]
	full := len(c.accounts.data.Users) >= 500
	c.accounts.mu.Unlock()
	if path == "/api/register" {
		if full {
			fail(w, 503, errors.New("New accounts are temporarily unavailable. You can continue as a guest."))
			return true
		}
		if exists {
			fail(w, 409, errors.New("That username is unavailable. Choose another or sign in."))
			return true
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
		if err != nil {
			fail(w, 500, errors.New("Unable to create account"))
			return true
		}
		u = account{Username: input.Username, PasswordHash: string(hash), Workspace: id()}
		// Copy the guest workspace, never adopt its cookie as an account credential.
		s, err := c.loadAccount(u.Workspace)
		if err != nil {
			fail(w, 500, errors.New("Unable to prepare workspace"))
			return true
		}
		if cookie, e := r.Cookie(cloudCookie); e == nil && cloudTokenPattern.MatchString(cookie.Value) {
			if guest, e := c.load(cookie.Value); e == nil {
				copy := guest.app.store.snapshot()
				copy.Messages = []Message{}
				if err = s.app.store.update(func(state *State) error { *state = copy; return nil }); err != nil {
					fail(w, 500, errors.New("Unable to copy guest workspace"))
					return true
				}
			}
		}
	} else {
		// A fixed-cost dummy hash prevents fast unknown-user responses.
		hash := u.PasswordHash
		if !exists {
			hash = "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW"
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)); err != nil || !exists {
			fail(w, 401, errors.New("Incorrect username or password."))
			return true
		}
	}
	token := id()
	c.accounts.mu.Lock()
	n := c.accounts.nextLocked()
	if path == "/api/register" {
		if _, ok := n.Users[name]; ok {
			c.accounts.mu.Unlock()
			fail(w, 409, errors.New("That username is unavailable. Choose another or sign in."))
			return true
		}
		if len(n.Users) >= 500 {
			c.accounts.mu.Unlock()
			fail(w, 503, errors.New("New accounts are temporarily unavailable. You can continue as a guest."))
			return true
		}
		n.Users[name] = u
	}
	if old, err := r.Cookie(accountCookie); err == nil {
		delete(n.Sessions, sessionDigest(old.Value))
	}
	// Bound per-account sessions without affecting other users.
	count := 0
	var oldest string
	var oldestTime time.Time
	for k, v := range n.Sessions {
		if v.User == name {
			count++
			if oldest == "" || v.Expires.Before(oldestTime) {
				oldest = k
				oldestTime = v.Expires
			}
		}
	}
	if count >= 10 {
		delete(n.Sessions, oldest)
	}
	n.Sessions[sessionDigest(token)] = accountSession{User: name, Expires: time.Now().Add(accountLifetime)}
	err := c.accounts.saveLocked(n)
	c.accounts.mu.Unlock()
	if err != nil {
		fail(w, 500, errors.New("Unable to save sign-in. Please try again."))
		return true
	}
	setAccountCookie(w, token, int(accountLifetime.Seconds()))
	clearGuestAccess(w)
	send(w, 200, map[string]any{"ok": true, "username": u.Username})
	return true
}
