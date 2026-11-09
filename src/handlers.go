package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("Invalid request format or size")
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("Unexpected trailing request data")
	}
	return nil
}
func keyValid(k string) bool { return len(k) <= 1024 && !strings.ContainsAny(k, "\r\n\x00") }
func (a *App) api(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/api/health" && r.Method == "GET" {
		send(w, 200, map[string]any{"app": "InvestOne", "version": Version, "mode": "LIVE_DATA_ONLY", "uptime_seconds": int(time.Since(a.boot).Seconds())})
		return
	}
	if r.Header.Get("X-InvestOne") != "1" {
		fail(w, 403, errors.New("Missing local-client verification header"))
		return
	}
	if path == "/api/login" && r.Method == "POST" {
		var v struct {
			Username string `json:"username"`
			Key      string `json:"key"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		if strings.TrimSpace(v.Username) != Username && !strings.EqualFold(strings.TrimSpace(v.Username), "RedBeanCoix") {
			fail(w, 401, errors.New("Sign in with RedBeanCoix or the original Chinese username"))
			return
		}
		v.Key = strings.TrimSpace(v.Key)
		if !keyValid(v.Key) {
			fail(w, 400, errors.New("Invalid API key format"))
			return
		}
		token := id()
		a.mu.Lock()
		for k, exp := range a.sessions {
			if time.Now().After(exp) {
				delete(a.sessions, k)
			}
		}
		if len(a.sessions) > 8 {
			a.sessions = map[string]time.Time{}
		}
		a.sessions[token] = time.Now().Add(7 * 24 * time.Hour)
		if v.Key != "" {
			a.key = v.Key
		}
		a.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "investone_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: os.Getenv("INVESTONE_CLOUD")=="1", MaxAge: 7 * 86400})
		send(w, 200, map[string]any{"ok": true, "ai_ready": a.getKey() != ""})
		return
	}
	if !a.authorized(r) {
		fail(w, 401, errors.New("Please sign in to InvestOne"))
		return
	}
	if path == "/api/diagnostics" && r.Method == "GET" {
		type check struct {
			Status     string `json:"status"`
			Source     string `json:"source,omitempty"`
			Error      string `json:"error,omitempty"`
			AsOf       string `json:"as_of,omitempty"`
			DurationMS int64  `json:"duration_ms"`
		}
		checks := map[string]check{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		run := func(label string, f func(context.Context) (string, string, error)) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				ctx, cancel := context.WithTimeout(r.Context(), 42*time.Second)
				defer cancel()
				source, asof, e := f(ctx)
				v := check{Status: "ok", Source: source, AsOf: asof, DurationMS: time.Since(start).Milliseconds()}
				if e != nil {
					v.Status = "failed"
					v.Error = e.Error()
				}
				mu.Lock()
				checks[label] = v
				mu.Unlock()
			}()
		}
		run("US stock/ETF history · SPY", func(ctx context.Context) (string, string, error) {
			c, e := a.provider.Market(ctx, "SPY", "1mo")
			return c.Source, c.AsOf, e
		})
		run("A-share history · 600519.SS", func(ctx context.Context) (string, string, error) {
			c, e := a.provider.Market(ctx, "600519.SS", "1mo")
			return c.Source, c.AsOf, e
		})
		run("Public financial news · NVDA", func(ctx context.Context) (string, string, error) {
			n, e := a.provider.News(ctx, "NVDA")
			if e != nil {
				return "", "", e
			}
			if len(n) == 0 {
				return "", "", errors.New("No items returned")
			}
			return n[0].Publisher, n[0].Date, nil
		})
		wg.Wait()
		mode := "Data-source connectivity diagnosis"
		if p, ok := a.provider.(*liveProvider); ok {
			mode = p.proxyMode
		}
		send(w, 200, map[string]any{"version": Version, "mode": mode, "checks": checks, "key_configured": a.getKey() != "", "timestamp": nowString(), "note": "Diagnostic requests do not change portfolio data. External providers may be unavailable; inspect the endpoint and error details."})
		return
	}
	if path == "/api/state" && r.Method == "GET" {
		send(w, 200, map[string]any{"state": a.store.snapshot(), "ai_ready": a.getKey() != "", "version": Version, "username": Username, "api_base": APIBase, "mode": "LIVE_DATA_ONLY"})
		return
	}
	if path == "/api/logout" && r.Method == "POST" {
		a.mu.Lock()
		a.sessions = map[string]time.Time{}
		a.key = ""
		a.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "investone_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: os.Getenv("INVESTONE_CLOUD")=="1", SameSite: http.SameSiteStrictMode})
		send(w, 200, map[string]bool{"ok": true})
		return
	}
	if path == "/api/export" && r.Method == "GET" {
		w.Header().Set("Content-Disposition", "attachment; filename=InvestOne-data.json")
		send(w, 200, a.store.snapshot())
		return
	}
	if path == "/api/quotes" && r.Method == "GET" {
		syms := strings.Split(r.URL.Query().Get("symbols"), ",")
		if len(syms) > 30 {
			fail(w, 400, errors.New("A maximum of 30 symbols per request is supported"))
			return
		}
		rng := r.URL.Query().Get("range")
		if rng == "" {
			rng = "1y"
		}
		quotes := []Chart{}
		errs := map[string]string{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		seen := map[string]bool{}
		for _, raw := range syms {
			sym, e := symbol(raw)
			if e != nil {
				mu.Lock()
				errs[raw] = e.Error()
				mu.Unlock()
				continue
			}
			if seen[sym] {
				continue
			}
			seen[sym] = true
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				c, e := a.provider.Market(r.Context(), s, rng)
				mu.Lock()
				defer mu.Unlock()
				if e != nil {
					errs[s] = e.Error()
				} else {
					quotes = append(quotes, c)
				}
			}(sym)
		}
		wg.Wait()
		sort.Slice(quotes, func(i, j int) bool { return quotes[i].Symbol < quotes[j].Symbol })
		send(w, 200, map[string]any{"quotes": quotes, "errors": errs})
		return
	}
	if path == "/api/research" && r.Method == "GET" {
		sym, e := symbol(r.URL.Query().Get("symbol"))
		if e != nil {
			fail(w, 400, e)
			return
		}
		email := a.store.snapshot().Profile.Email
		result := map[string]any{"symbol": sym, "sec_url": "https://www.sec.gov/edgar/search/#/q=" + url.QueryEscape(sym)}
		errs := map[string]string{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		tasks := map[string]func() (any, error){"news": func() (any, error) { return a.provider.News(r.Context(), sym) }}
		if validEmail(email) {
			tasks["filings"] = func() (any, error) { return a.provider.Filings(r.Context(), sym, email) }
			tasks["facts"] = func() (any, error) { return a.provider.Facts(r.Context(), sym, email) }
		} else {
			errs["sec"] = "SEC API requests require a contact email in Settings; official filing search links remain available"
		}
		for k, f := range tasks {
			wg.Add(1)
			go func(k string, f func() (any, error)) {
				defer wg.Done()
				v, e := f()
				mu.Lock()
				defer mu.Unlock()
				if e != nil {
					errs[k] = e.Error()
				} else {
					result[k] = v
				}
			}(k, f)
		}
		wg.Wait()
		result["errors"] = errs
		send(w, 200, result)
		return
	}
	if path == "/api/community" && r.Method == "GET" {
		sym, e := symbol(r.URL.Query().Get("symbol"))
		if e != nil {
			fail(w, 400, e)
			return
		}
		items, e := a.provider.Community(r.Context(), sym)
		msg := ""
		if e != nil {
			msg = e.Error()
		}
		if items == nil {
			items = []Source{}
		}
		send(w, 200, map[string]any{"items": items, "saved": a.store.snapshot().Community, "error": msg})
		return
	}
	if r.Method != "POST" {
		fail(w, 404, errors.New("Endpoint not found"))
		return
	}
	switch path {
	case "/api/pdf-text":
		a.pdfText(w, r)
	case "/api/key":
		var v struct {
			Key string `json:"key"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		v.Key = strings.TrimSpace(v.Key)
		if !keyValid(v.Key) {
			fail(w, 400, errors.New("Invalid API key format"))
			return
		}
		a.mu.Lock()
		a.key = v.Key
		a.mu.Unlock()
		send(w, 200, map[string]bool{"ok": true, "ai_ready": v.Key != ""})
	case "/api/profile":
		var p Profile
		if e := decode(w, r, &p); e != nil {
			fail(w, 400, e)
			return
		}
		if p.Email != "" && !validEmail(p.Email) {
			fail(w, 400, errors.New("Invalid contact email format"))
			return
		}
		if p.MaxTokens < 256 || p.MaxTokens > 65536 || len(p.Model) < 1 || len(p.Model) > 160 || len(p.Interests) > 3000 || len(p.Risk) > 100 || len(p.Horizon) > 100 || len(p.Knowledge) > 100 {
			fail(w, 400, errors.New("Invalid configuration; output tokens must be between 256 and 65536"))
			return
		}
		e := a.store.update(func(s *State) error { s.Profile = p; return nil })
		respondUpdate(w, e)
	case "/api/watchlist":
		var v struct {
			Symbol string `json:"symbol"`
			Remove bool   `json:"remove"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		sym, e := symbol(v.Symbol)
		if e != nil {
			fail(w, 400, e)
			return
		}
		e = a.store.update(func(s *State) error {
			idx := -1
			for i, x := range s.Watchlist {
				if x == sym {
					idx = i
					break
				}
			}
			if v.Remove {
				if idx >= 0 {
					s.Watchlist = append(s.Watchlist[:idx], s.Watchlist[idx+1:]...)
				}
			} else if idx < 0 {
				if len(s.Watchlist) >= 30 {
					return errors.New("Watchlist limit is 30 symbols")
				}
				s.Watchlist = append(s.Watchlist, sym)
			}
			return nil
		})
		respondUpdate(w, e)
	case "/api/trade":
		var v struct {
			Symbol   string  `json:"symbol"`
			Side     string  `json:"side"`
			Quantity float64 `json:"quantity"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		sym, e := symbol(v.Symbol)
		if e != nil {
			fail(w, 400, e)
			return
		}
		ch, e := a.provider.Market(r.Context(), sym, "1mo")
		if e != nil {
			fail(w, 502, e)
			return
		}
		if ch.Currency != "USD" || strings.HasPrefix(sym, "^") {
			fail(w, 400, errors.New("Paper account supports USD-denominated stocks or ETFs only"))
			return
		}
		asof, e := time.Parse(time.RFC3339, ch.AsOf)
		if e != nil || time.Since(asof) > 5*24*time.Hour || asof.After(time.Now().Add(24*time.Hour)) {
			fail(w, 400, errors.New("Quote date is missing or out of range; paper trade was not executed"))
			return
		}
		e = a.store.update(func(s *State) error { return applyTrade(s, v.Side, sym, v.Quantity, ch.Price, ch.Source, ch.AsOf) })
		respondUpdate(w, e)
	case "/api/chat":
		var v ChatRequest
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		c, cancel := context.WithTimeout(r.Context(), 200*time.Second)
		defer cancel()
		res, e := a.chat(c, v, true)
		if e != nil {
			fail(w, 502, e)
			return
		}
		send(w, 200, res)
	case "/api/quant":
		var v QuantConfig
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		res, e := a.runQuant(r.Context(), v)
		if e != nil {
			fail(w, 422, e)
			return
		}
		send(w, 200, res)
	case "/api/documents":
		var v struct {
			Name string `json:"name"`
			Text string `json:"text"`
			URL  string `json:"url"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		v.Name = strings.TrimSpace(v.Name)
		if len(v.Name) < 1 || len(v.Name) > 200 || len([]rune(v.Text)) < 10 || len([]rune(v.Text)) > 400000 {
			fail(w, 400, errors.New("Documents need a valid title and 10–400,000 characters"))
			return
		}
		if v.URL != "" && safeURL(v.URL) == "" {
			fail(w, 400, errors.New("Source URL must be a valid HTTPS link"))
			return
		}
		e := a.store.update(func(s *State) error {
			if len(s.Documents) >= 40 {
				return errors.New("Local library limit is 40 documents")
			}
			s.Documents = append(s.Documents, Document{ID: id(), Name: v.Name, Text: v.Text, URL: v.URL, Created: nowString()})
			return nil
		})
		respondUpdate(w, e)
	case "/api/documents/delete":
		var v struct {
			ID string `json:"id"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		e := a.store.update(func(s *State) error {
			for i, d := range s.Documents {
				if d.ID == v.ID {
					s.Documents = append(s.Documents[:i], s.Documents[i+1:]...)
					return nil
				}
			}
			return errors.New("Document not found")
		})
		respondUpdate(w, e)
	case "/api/filing/import":
		var v struct {
			Symbol string `json:"symbol"`
			URL    string `json:"url"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		sym, e := symbol(v.Symbol)
		if e != nil {
			fail(w, 400, e)
			return
		}
		email := a.store.snapshot().Profile.Email
		items, e := a.provider.Filings(r.Context(), sym, email)
		if e != nil {
			fail(w, 400, e)
			return
		}
		found := ""
		for _, s := range items {
			if s.URL == v.URL {
				found = s.Title
				break
			}
		}
		if found == "" {
			fail(w, 400, errors.New("Import is supported only from the current SEC filing list"))
			return
		}
		p, ok := a.provider.(*liveProvider)
		if !ok {
			fail(w, 501, errors.New("This test provider does not support importing original documents"))
			return
		}
		b, e := p.fetch(r.Context(), v.URL, "InvestOne "+email, 24*time.Hour)
		if e != nil {
			fail(w, 502, e)
			return
		}
		text := plain(string(b))
		if strings.HasPrefix(string(b), "%PDF") || len([]rune(text)) < 100 {
			fail(w, 422, errors.New("This file is not supported HTML; open the original filing instead"))
			return
		}
		text = trimRunes(text, 400000)
		e = a.store.update(func(s *State) error {
			if len(s.Documents) >= 40 {
				return errors.New("Research library is full")
			}
			s.Documents = append(s.Documents, Document{ID: id(), Name: found, Text: text, URL: v.URL, Created: nowString()})
			return nil
		})
		respondUpdate(w, e)
	case "/api/community/save":
		var v struct {
			Title string `json:"title"`
			Text  string `json:"text"`
			URL   string `json:"url"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		if len(v.Title) < 1 || len(v.Title) > 200 || len(v.Text) < 1 || len(v.Text) > 12000 {
			fail(w, 400, errors.New("Enter a title and public opinion text (up to 12,000 characters)"))
			return
		}
		if v.URL != "" && safeURL(v.URL) == "" {
			fail(w, 400, errors.New("Source URL must use HTTPS"))
			return
		}
		e := a.store.update(func(s *State) error {
			if len(s.Community) >= 200 {
				return errors.New("Saved public opinions are limited to 200")
			}
			s.Community = append(s.Community, Source{ID: id(), Title: v.Title, Text: v.Text, URL: v.URL, Publisher: "User-saved public opinion (unverified)", Date: nowString(), Kind: "opinion"})
			return nil
		})
		respondUpdate(w, e)
	case "/api/rules":
		var v struct {
			Type      string  `json:"type"`
			Symbol    string  `json:"symbol"`
			Threshold float64 `json:"threshold"`
			Keyword   string  `json:"keyword"`
			AutoAI    bool    `json:"auto_ai"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		sym, e := symbol(v.Symbol)
		if e != nil {
			fail(w, 400, e)
			return
		}
		allowed := map[string]bool{"price_below": true, "price_above": true, "change_above": true, "change_below": true, "keyword": true, "filing": true, "weekly": true}
		if !allowed[v.Type] || !finite(v.Threshold) || len(v.Keyword) > 200 {
			fail(w, 400, errors.New("Invalid monitoring rule parameters"))
			return
		}
		if strings.HasPrefix(v.Type, "price_") && v.Threshold <= 0 {
			fail(w, 400, errors.New("Price threshold must be greater than zero"))
			return
		}
		if v.Type == "keyword" && strings.TrimSpace(v.Keyword) == "" {
			fail(w, 400, errors.New("Enter a news keyword"))
			return
		}
		if v.Type == "filing" && !validEmail(a.store.snapshot().Profile.Email) {
			fail(w, 400, errors.New("Filing alerts require an SEC contact email in Settings"))
			return
		}
		e = a.store.update(func(s *State) error {
			if len(s.Rules) >= 30 {
				return errors.New("Monitoring rule limit is 30")
			}
			s.Rules = append(s.Rules, Rule{ID: id(), Type: v.Type, Symbol: sym, Threshold: v.Threshold, Keyword: v.Keyword, AutoAI: v.AutoAI, Enabled: true, Created: nowString()})
			return nil
		})
		respondUpdate(w, e)
	case "/api/rules/update":
		var v struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
			Delete  bool   `json:"delete"`
		}
		if e := decode(w, r, &v); e != nil {
			fail(w, 400, e)
			return
		}
		e := a.store.update(func(s *State) error {
			for i := range s.Rules {
				if s.Rules[i].ID == v.ID {
					if v.Delete {
						s.Rules = append(s.Rules[:i], s.Rules[i+1:]...)
					} else {
						s.Rules[i].Enabled = v.Enabled
					}
					return nil
				}
			}
			return errors.New("Monitoring rule not found")
		})
		respondUpdate(w, e)
	case "/api/rules/check":
		n := a.checkRules(r.Context())
		if n < 0 {
			fail(w, 409, errors.New("A monitoring check is already running"))
			return
		}
		send(w, 200, map[string]any{"ok": true, "triggered": n})
	case "/api/notices/read":
		e := a.store.update(func(s *State) error {
			for i := range s.Notices {
				s.Notices[i].Read = true
			}
			return nil
		})
		respondUpdate(w, e)
	default:
		fail(w, 404, fmt.Errorf("Endpoint not found"))
	}
}
func respondUpdate(w http.ResponseWriter, e error) {
	if e != nil {
		fail(w, 400, e)
	} else {
		send(w, 200, map[string]bool{"ok": true})
	}
}
