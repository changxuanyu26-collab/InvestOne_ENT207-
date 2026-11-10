package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const cloudCookie = "investone_browser"

var cloudTokenPattern = regexp.MustCompile(`^[a-f0-9]{48}$`)

// cloudSession holds one independently persisted browser's InvestOne application.
// No model API keys are written to disk. Cookies are random, HttpOnly and Secure.
type cloudSession struct {
	app     *App
	cancel  context.CancelFunc
	mu      sync.Mutex
	started time.Time
	hits    int
	aiStarted time.Time
	aiHits int
}

type cloudServer struct {
	mu       sync.Mutex
	sessions map[string]*cloudSession
	provider Provider
	dataDir  string
	ctx      context.Context
	static   http.Handler
	visits   *visitTracker
	aiGate   chan struct{}
}

func newCloudServer(ctx context.Context, root string) (*cloudServer, error) {
	if root == "" {
		return nil, errors.New("missing cloud data directory")
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		return nil, e
	}
	sub, e := fs.Sub(webFiles, "web")
	if e != nil {
		return nil, e
	}
	c := &cloudServer{sessions: map[string]*cloudSession{}, provider: newLiveProvider(), dataDir: root, ctx: ctx, static: http.FileServer(http.FS(sub)), visits: newVisitTracker(root), aiGate: make(chan struct{}, 2)}
	if key := strings.TrimSpace(os.Getenv("INVESTONE_SHARED_AI_KEY")); key != "" && !keyValid(key) {
		return nil, errors.New("invalid server-side AI key configuration")
	}
	// Recover saved browser workspaces after a restart so monitoring can continue.
	files, e := os.ReadDir(root)
	if e != nil {
		return nil, e
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		token := strings.TrimSuffix(f.Name(), ".json")
		if !cloudTokenPattern.MatchString(token) {
			continue
		}
		if len(c.sessions) >= 500 {
			break
		}
		if _, err := c.load(token); err != nil {
			fmt.Fprintf(os.Stderr, "Skipped damaged workspace: %v\n", err)
		}
	}
	return c, nil
}
func (c *cloudServer) load(token string) (*cloudSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.sessions[token]; ok {
		return existing, nil
	}
	path := filepath.Join(c.dataDir, token+".json")
	store, e := openStore(path)
	if e != nil {
		return nil, e
	}
	app := newApp(store, c.provider)
	ctx, cancel := context.WithCancel(c.ctx)
	sess := &cloudSession{app: app, cancel: cancel, started: time.Now()}
	c.sessions[token] = sess
	go app.monitorLoop(ctx)
	return sess, nil
}
func (c *cloudServer) browserSession(w http.ResponseWriter, r *http.Request) (*cloudSession, error) {
	token := ""
	if cookie, e := r.Cookie(cloudCookie); e == nil && cloudTokenPattern.MatchString(cookie.Value) {
		token = cookie.Value
	}
	if token == "" {
		token = id()
		http.SetCookie(w, &http.Cookie{Name: cloudCookie, Value: token, Path: "/", MaxAge: 30 * 86400, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		c.visits.browserSession()
	}
	return c.load(token)
}
func (s *cloudSession) withinLimit() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if now.Sub(s.started) > time.Minute {
		s.started = now
		s.hits = 0
	}
	s.hits++
	return s.hits <= 150
}
func (s *cloudSession) allowAI() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if now.Sub(s.aiStarted) > 10*time.Minute {
		s.aiStarted = now
		s.aiHits = 0
	}
	if s.aiHits >= 6 {
		return false
	}
	s.aiHits++
	return true
}
func (c *cloudServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		// Browsers correctly label navigation *from the GitHub Pages landing*
		// as cross-site. Permit only a safe, top-level HTML navigation to
		// the public landing/app routes. Keep cross-site API calls, embeds,
		// resource requests and unsafe methods blocked (CSRF protection).
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			navigation := (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
				r.Header.Get("Sec-Fetch-Mode") == "navigate" &&
				r.Header.Get("Sec-Fetch-Dest") == "document"
			publicPage := r.URL.Path == "/" || r.URL.Path == "/landing" ||
				r.URL.Path == "/app" || r.URL.Path == "/app/" || r.URL.Path == "/index.html"
			if !navigation || !publicPage {
				http.Error(w, "Cross-site request blocked", http.StatusForbidden)
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if r.URL.Path == "/api/health" && r.Method == http.MethodGet {
				send(w, 200, map[string]any{"app": "InvestOne", "version": Version, "mode": "CLOUD_LIVE_BACKEND", "status": "ok"})
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", 405)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				scheme := "https"
				if r.TLS != nil {
					scheme = "https"
				} else if r.Header.Get("X-Forwarded-Proto") == "http" {
					scheme = "http"
				}
				expected := scheme + "://" + r.Host
				if subtle.ConstantTimeCompare([]byte(origin), []byte(expected)) != 1 {
					http.Error(w, "Origin mismatch", 403)
					return
				}
			}
			if len(r.Header.Get("X-InvestOne")) == 0 {
				http.Error(w, "Missing application request header", 403)
				return
			}
			if r.URL.Path == "/api/visits" && r.Method == http.MethodGet {
				send(w, 200, c.visits.snapshot())
				return
			}
			s, e := c.browserSession(w, r)
			if e != nil {
				fail(w, 500, errors.New("Workspace storage unavailable"))
				return
			}
			if !s.withinLimit() {
				fail(w, 429, errors.New("Too many requests; try again shortly"))
				return
			}
			if r.Method == http.MethodPost && (r.URL.Path == "/api/chat" || r.URL.Path == "/api/chat/stream") && s.app.sharedKey != "" {
				select {
				case c.aiGate <- struct{}{}:
					defer func() { <-c.aiGate }()
				default:
					fail(w, 429, errors.New("The public AI service is busy; try again shortly"))
					return
				}
				if !s.allowAI() {
					fail(w, 429, errors.New("This browser has reached its AI request limit; try again in 10 minutes"))
					return
				}
				if !c.visits.allowAI() {
					fail(w, 429, errors.New("The shared daily AI quota has been reached; try again tomorrow"))
					return
				}
			}
			// A first-time visitor can explore real financial data without an
			// account. The public deployment can use a server-managed AI key.
			if r.URL.Path == "/api/state" && r.Method == http.MethodGet && !s.app.authorized(r) {
				token := id()
				s.app.mu.Lock()
				s.app.sessions[token] = time.Now().Add(7 * 24 * time.Hour)
				s.app.mu.Unlock()
				http.SetCookie(w, &http.Cookie{Name: "investone_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: 7 * 86400})
				r.AddCookie(&http.Cookie{Name: "investone_session", Value: token})
			}
			s.app.api(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", 405)
			return
		}
		switch r.URL.Path {
		case "/", "/landing":
			b, e := webFiles.ReadFile("web/landing.html")
			if e != nil {
				http.Error(w, "Landing page unavailable", 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(b)
		case "/app", "/app/", "/index.html":
			if r.Method == http.MethodGet {
				c.visits.pageView()
			}
			b, e := webFiles.ReadFile("web/index.html")
			if e != nil {
				http.Error(w, "App page unavailable", 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(b)
		case "/app.js", "/styles.css", "/favicon.svg":
			c.static.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}
