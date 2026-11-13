package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const accountCookie = "investone_account" // Read-only compatibility with the previous release.
const guestAccessCookie = "investone_guest_access"
const namedAccessCookie = "investone_named_access"

var errExpiredAccount = errors.New("Please enter your name to continue.")

type account struct {
	Username  string `json:"username"`
	Workspace string `json:"workspace"`
}
type accountSession struct {
	User    string    `json:"user"`
	Expires time.Time `json:"expires"`
}
type accountData struct {
	Users    map[string]account        `json:"users"`
	Sessions map[string]accountSession `json:"sessions"`
}
type accountStore struct {
	mu    sync.Mutex
	path  string
	data  accountData // Legacy records are read only; never unlock them by display name.
	names map[string]account
}

func openAccounts(root string) (*accountStore, error) {
	a := &accountStore{path: filepath.Join(root, "browser-names.json"), names: map[string]account{}, data: accountData{Users: map[string]account{}, Sessions: map[string]accountSession{}}}
	if b, err := os.ReadFile(filepath.Join(root, "accounts.json")); err == nil {
		if json.Unmarshal(b, &a.data) != nil {
			return nil, errors.New("Invalid legacy workspace records")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if b, err := os.ReadFile(a.path); err == nil {
		if json.Unmarshal(b, &a.names) != nil || a.names == nil {
			return nil, errors.New("Invalid browser workspace records")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for token, n := range a.names {
		if !cloudTokenPattern.MatchString(token) || !validDisplayName(n.Username) || (n.Workspace != "" && !cloudTokenPattern.MatchString(n.Workspace)) {
			return nil, errors.New("Invalid browser workspace records")
		}
	}
	return a, nil
}
func validDisplayName(name string) bool {
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 60 || strings.TrimSpace(name) == "" {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func sessionDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
	return u, ok && cloudTokenPattern.MatchString(u.Workspace)
}
func (a *accountStore) remember(token, name, workspace string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	next := make(map[string]account, len(a.names)+1)
	for k, v := range a.names {
		next[k] = v
	}
	n := next[token]
	n.Username = name
	if workspace != "" {
		n.Workspace = workspace
	}
	next[token] = n
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(a.path), ".names-*.tmp")
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
	closed := f.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	if err = os.Rename(f.Name(), a.path); err != nil {
		return err
	}
	a.names = next
	return nil
}
func browserToken(r *http.Request) string {
	if cookie, err := r.Cookie(cloudCookie); err == nil && cloudTokenPattern.MatchString(cookie.Value) {
		return cookie.Value
	}
	return ""
}
func (c *cloudServer) identity(r *http.Request) (account, bool) {
	active, err := r.Cookie(namedAccessCookie)
	if err != nil || active.Value != "1" {
		return account{}, false
	}
	token := browserToken(r)
	if token == "" {
		return account{}, false
	}
	c.accounts.mu.Lock()
	defer c.accounts.mu.Unlock()
	n, ok := c.accounts.names[token]
	return n, ok
}
func setAccessCookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (c *cloudServer) workspace(w http.ResponseWriter, r *http.Request) (*cloudSession, *account, error) {
	if n, ok := c.identity(r); ok {
		if n.Workspace != "" {
			s, err := c.loadAccount(n.Workspace)
			return s, &n, err
		}
		s, err := c.browserSession(w, r)
		return s, &n, err
	}
	// Existing signed-in browsers may still recover their previous workspace.
	if cookie, err := r.Cookie(accountCookie); err == nil && cookie.Value != "" {
		if n, ok := c.accounts.resolve(cookie.Value); ok {
			s, err := c.loadAccount(n.Workspace)
			return s, &n, err
		}
	}
	s, err := c.browserSession(w, r)
	return s, nil, err
}
func (c *cloudServer) auth(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path != "/api/auth/session" && path != "/api/login" && path != "/api/logout" && path != "/api/register" && path != "/api/guest" {
		return false
	}
	if path == "/api/auth/session" {
		if r.Method != "GET" {
			fail(w, 405, errors.New("Method not allowed"))
			return true
		}
		n, ok := c.identity(r)
		// Migrate an already signed-in browser using its existing credential, not its name.
		if !ok {
			if cookie, err := r.Cookie(accountCookie); err == nil {
				if old, valid := c.accounts.resolve(cookie.Value); valid {
					token := browserToken(r)
					if token == "" {
						token = id()
						setAccessCookie(w, cloudCookie, token, 30*86400)
					}
					if err := c.accounts.remember(token, old.Username, old.Workspace); err != nil {
						fail(w, 500, errors.New("Unable to restore workspace"))
						return true
					}
					setAccessCookie(w, namedAccessCookie, "1", 30*86400)
					n = old
					ok = true
				}
				setAccessCookie(w, accountCookie, "", -1)
			}
		}
		send(w, 200, map[string]any{"cloud": true, "signed_in": ok, "authenticated": false, "username": n.Username, "guest": false})
		return true
	}
	if r.Method != "POST" {
		fail(w, 405, errors.New("Method not allowed"))
		return true
	}
	if path == "/api/register" {
		fail(w, 410, errors.New("Registration is no longer needed. Reload and enter your name."))
		return true
	}
	if path == "/api/logout" || path == "/api/guest" {
		if cookie, err := r.Cookie("investone_session"); err == nil {
			if s, _, e := c.workspace(w, r); e == nil {
				s.app.mu.Lock()
				delete(s.app.sessions, cookie.Value)
				s.app.mu.Unlock()
			}
		}
		setAccessCookie(w, namedAccessCookie, "", -1)
		setAccessCookie(w, accountCookie, "", -1)
		setAccessCookie(w, guestAccessCookie, "", -1)
		http.SetCookie(w, &http.Cookie{Name: "investone_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		send(w, 200, map[string]bool{"ok": true})
		return true
	}
	var input struct {
		Username string `json:"username"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := decode(w, r, &input); err != nil {
		fail(w, 400, err)
		return true
	}
	input.Username = strings.TrimSpace(input.Username)
	if !validDisplayName(input.Username) {
		fail(w, 400, errors.New("Enter a name of 1–60 characters."))
		return true
	}
	token := browserToken(r)
	if token == "" {
		token = id()
	}
	workspace := ""
	if old, err := r.Cookie(accountCookie); err == nil {
		if n, ok := c.accounts.resolve(old.Value); ok {
			workspace = n.Workspace
		}
	}
	if err := c.accounts.remember(token, input.Username, workspace); err != nil {
		fail(w, 500, errors.New("Unable to save your name. Please try again."))
		return true
	}
	setAccessCookie(w, cloudCookie, token, 30*86400)
	setAccessCookie(w, namedAccessCookie, "1", 30*86400)
	setAccessCookie(w, accountCookie, "", -1)
	setAccessCookie(w, guestAccessCookie, "", -1)
	send(w, 200, map[string]any{"ok": true, "username": input.Username})
	return true
}
