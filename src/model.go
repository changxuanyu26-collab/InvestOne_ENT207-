package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const Version = "2.1.0-rc1"
const DefaultModel = "DeepSeek-V4-Flash"
const APIBase = "https://apim-ent207tc.azure-api.net/deepseek/openai/v1"
const Username = "红豆薏米水"
const InitialCash = 100000.0
const PaperFee = 0.0005

type Profile struct {
	Risk      string `json:"risk"`
	Horizon   string `json:"horizon"`
	Knowledge string `json:"knowledge"`
	Interests string `json:"interests"`
	Email     string `json:"email"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}
type Holding struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Cost     float64 `json:"cost"`
}
type Trade struct {
	ID       string  `json:"id"`
	Time     string  `json:"time"`
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Fee      float64 `json:"fee"`
	Realized float64 `json:"realized"`
	Source   string  `json:"source"`
	AsOf     string  `json:"as_of"`
}
type Source struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Publisher string `json:"publisher"`
	Date      string `json:"date"`
	Text      string `json:"text,omitempty"`
	Kind      string `json:"kind"`
}
type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Time    string   `json:"time"`
	Persona string   `json:"persona,omitempty"`
	Sources []Source `json:"sources,omitempty"`
}
type Document struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Text    string `json:"text"`
	URL     string `json:"url"`
	Created string `json:"created"`
}
type Rule struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	Symbol        string  `json:"symbol"`
	Threshold     float64 `json:"threshold"`
	Keyword       string  `json:"keyword"`
	Enabled       bool    `json:"enabled"`
	AutoAI        bool    `json:"auto_ai"`
	Latched       bool    `json:"latched"`
	LastKey       string  `json:"last_key"`
	LastCheck     string  `json:"last_check"`
	LastTriggered string  `json:"last_triggered"`
	Error         string  `json:"error"`
	Created       string  `json:"created"`
}
type Notice struct {
	ID     string `json:"id"`
	Time   string `json:"time"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Read   bool   `json:"read"`
	RuleID string `json:"rule_id"`
}
type State struct {
	Schema    int        `json:"schema"`
	Profile   Profile    `json:"profile"`
	Watchlist []string   `json:"watchlist"`
	Cash      float64    `json:"cash"`
	Holdings  []Holding  `json:"holdings"`
	Trades    []Trade    `json:"trades"`
	Messages  []Message  `json:"messages"`
	Documents []Document `json:"documents"`
	Rules     []Rule     `json:"rules"`
	Notices   []Notice   `json:"notices"`
	Community []Source   `json:"community"`
}

type Store struct {
	mu    sync.Mutex
	path  string
	value State
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339) }
func id() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func initialState() State {
	return State{Schema: 1, Profile: Profile{Risk: "Moderate", Horizon: "3–5 years", Knowledge: "Beginner", Interests: "Technology, advanced manufacturing, sector allocation", Model: DefaultModel, MaxTokens: 400}, Watchlist: []string{"AAPL", "NVDA", "MSFT", "SPY", "QQQ"}, Cash: InitialCash, Holdings: []Holding{}, Trades: []Trade{}, Messages: []Message{}, Documents: []Document{}, Rules: []Rule{}, Notices: []Notice{}, Community: []Source{}}
}
func openStore(path string) (*Store, error) {
	s := &Store{path: path, value: initialState()}
	b, e := os.ReadFile(path)
	if e == nil {
		if e = json.Unmarshal(b, &s.value); e != nil {
			return nil, fmt.Errorf("Unable to read local data; original file remains unchanged: %w", e)
		}
		if s.value.Schema != 1 {
			return nil, errors.New("Unsupported local data version; original file has not been overwritten")
		}
		// Prior local and cloud workspaces keep their user data while the
		// ENT207 course model is switched to the gateway's exact model ID.
		s.value.Profile.Model = DefaultModel
		if s.value.Profile.MaxTokens > 700 || s.value.Profile.MaxTokens < 128 {
			s.value.Profile.MaxTokens = 400
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	return s, nil
}
func clone(s State) State        { b, _ := json.Marshal(s); var c State; _ = json.Unmarshal(b, &c); return c }
func (s *Store) snapshot() State { s.mu.Lock(); defer s.mu.Unlock(); return clone(s.value) }
func (s *Store) update(f func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := clone(s.value)
	if e := f(&n); e != nil {
		return e
	}
	if s.path != "" {
		if e := os.MkdirAll(filepath.Dir(s.path), 0700); e != nil {
			return e
		}
		b, e := json.MarshalIndent(n, "", "  ")
		if e != nil {
			return e
		}
		t, e := os.CreateTemp(filepath.Dir(s.path), ".investone-*.tmp")
		if e != nil {
			return e
		}
		p := t.Name()
		defer os.Remove(p)
		_ = t.Chmod(0600)
		if _, e = t.Write(b); e == nil {
			e = t.Sync()
		}
		ce := t.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		if e = os.Rename(p, s.path); e != nil {
			return e
		}
	}
	s.value = n
	return nil
}

var symbolPattern = regexp.MustCompile(`^[A-Z0-9^][A-Z0-9.^=\-]{0,19}$`)

func symbol(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !symbolPattern.MatchString(s) {
		return "", errors.New("Enter a valid stock symbol, e.g., NVDA, 0700.HK or 600519.SS")
	}
	return s, nil
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func trimRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func applyTrade(s *State, side, sym string, qty, price float64, source, asof string) error {
	if !finite(qty) || !finite(price) || qty <= 0 || qty > 1e8 || price <= 0 || price > 1e8 {
		return errors.New("Quantity and price must be finite positive numbers")
	}
	if side != "buy" && side != "sell" {
		return errors.New("Unsupported paper-trade side")
	}
	amount := qty * price
	fee := amount * PaperFee
	idx := -1
	for i, h := range s.Holdings {
		if h.Symbol == sym {
			idx = i
			break
		}
	}
	realized := 0.0
	if side == "buy" {
		if amount+fee > s.Cash+1e-8 {
			return errors.New("Insufficient paper cash including the 5 bps simulated fee")
		}
		s.Cash -= amount + fee
		if idx < 0 {
			s.Holdings = append(s.Holdings, Holding{Symbol: sym, Quantity: qty, Cost: (amount + fee) / qty})
		} else {
			h := &s.Holdings[idx]
			h.Cost = (h.Cost*h.Quantity + amount + fee) / (h.Quantity + qty)
			h.Quantity += qty
		}
	} else {
		if idx < 0 || qty > s.Holdings[idx].Quantity+1e-8 {
			return errors.New("Insufficient paper holdings; short-selling is not supported")
		}
		h := &s.Holdings[idx]
		realized = (price-h.Cost)*qty - fee
		s.Cash += amount - fee
		h.Quantity -= qty
		if h.Quantity < 1e-8 {
			s.Holdings = append(s.Holdings[:idx], s.Holdings[idx+1:]...)
		}
	}
	s.Trades = append(s.Trades, Trade{ID: id(), Time: nowString(), Symbol: sym, Side: side, Quantity: qty, Price: price, Fee: fee, Realized: realized, Source: source, AsOf: asof})
	return nil
}
