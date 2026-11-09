package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed web/*
var webFiles embed.FS

type App struct {
	store       *Store
	provider    Provider
	client      *http.Client
	aiURL       string
	mu          sync.Mutex
	sessions    map[string]time.Time
	key         string
	aiGate      chan struct{}
	quantGate   chan struct{}
	monitorGate chan struct{}
	boot        time.Time
}

func aiTransport() *http.Transport { t, _ := configuredTransport(); return t }
func newApp(store *Store, p Provider) *App {
	return &App{store: store, provider: p, client: &http.Client{Transport: aiTransport(), Timeout: 180 * time.Second, CheckRedirect: func(r *http.Request, v []*http.Request) error { return http.ErrUseLastResponse }}, aiURL: APIBase + "/chat/completions", sessions: map[string]time.Time{}, aiGate: make(chan struct{}, 1), quantGate: make(chan struct{}, 1), monitorGate: make(chan struct{}, 1), boot: time.Now()}
}
func (a *App) getKey() string { a.mu.Lock(); defer a.mu.Unlock(); return a.key }
func (a *App) handler() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFiles, "web")
	static := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			a.api(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "method not allowed", 405)
			return
		}
		static.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		host, _, e := net.SplitHostPort(r.Host)
		if e != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "Only loopback host is allowed", 403)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+r.Host {
			http.Error(w, "Cross-origin access blocked", 403)
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "Cross-site access blocked", 403)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
func send(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, e error) {
	send(w, status, map[string]string{"error": e.Error()})
}
func (a *App) authorized(r *http.Request) bool {
	c, e := r.Cookie("investone_session")
	if e != nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[c.Value]
	return ok && time.Now().Before(exp)
}
func dataPath() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "InvestOne", "data", "investone.json")
	}
	return filepath.Join(home, ".local", "share", "InvestOne", "investone.json")
}
func main() {
	addr := flag.String("addr", "127.0.0.1:18791", "loopback address")
	data := flag.String("data", dataPath(), "local data file")
	browser := flag.Bool("open", runtime.GOOS == "darwin", "open browser")
	version := flag.Bool("version", false, "print version")
	flag.Parse()
	if *version {
		fmt.Println("InvestOne", Version, runtime.GOOS, runtime.GOARCH)
		return
	}
	host, _, e := net.SplitHostPort(*addr)
	if e != nil || host != "127.0.0.1" {
		log.Fatal("Only localhost (127.0.0.1) binding is allowed")
	}
	store, e := openStore(*data)
	if e != nil {
		log.Fatal(e)
	}
	a := newApp(store, newLiveProvider())
	server := &http.Server{Addr: *addr, Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 330 * time.Second, IdleTimeout: 60 * time.Second}
	l, e := net.Listen("tcp", *addr)
	if e != nil {
		c := &http.Client{Timeout: 2 * time.Second}
		if resp, err := c.Get("http://" + *addr + "/api/health"); err == nil {
			var health struct {
				App string `json:"app"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&health)
			_ = resp.Body.Close()
			if health.App == "InvestOne" {
				if *browser && runtime.GOOS == "darwin" {
					_ = exec.Command("/usr/bin/open", "http://"+*addr).Run()
				}
				return
			}
		}
		log.Fatal("Startup failed; port may be occupied: ", e)
	}
	pidPath := filepath.Join(filepath.Dir(filepath.Dir(*data)), "investone.pid")
	if err := os.MkdirAll(filepath.Dir(pidPath), 0700); err == nil {
		_ = os.WriteFile(pidPath, []byte(fmt.Sprint(os.Getpid())), 0600)
		defer os.Remove(pidPath)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go a.monitorLoop(ctx)
	if *browser && runtime.GOOS == "darwin" {
		go func() { time.Sleep(300 * time.Millisecond); _ = exec.Command("/usr/bin/open", "http://"+*addr).Run() }()
	}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(c)
	}()
	log.Printf("InvestOne %s listening at http://%s (local only)", Version, *addr)
	if e = server.Serve(l); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
