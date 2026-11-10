package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixtureProvider struct {
	fail     bool
	price    float64
	currency string
	date     string
}

func testChart(s string) Chart {
	seed := 0
	for _, r := range s {
		seed += int(r)
	}
	ps := []Point{}
	start := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	v := 100.
	for d, i := start, 0; i < 540; d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		v *= 1 + .0004 + float64(seed%17)/30000 + .008*math.Sin(float64(i)/8+float64(seed))
		ps = append(ps, Point{Date: d.Format("2006-01-02"), Close: v})
		i++
	}
	return Chart{Symbol: s, Name: s + " [TEST FIXTURE]", Currency: "USD", Price: 150, Previous: 148, Change: (150./148 - 1) * 100, AsOf: nowString(), Retrieved: nowString(), Source: "AUTOMATED_TEST_FIXTURE_NOT_LIVE_MARKET_DATA", URL: "https://example.com/fixture/" + s, Adjusted: true, Points: ps}
}
func (p *fixtureProvider) Market(ctx context.Context, s, r string) (Chart, error) {
	if p.fail {
		return Chart{}, errors.New("FIXTURE_PROVIDER_UNAVAILABLE")
	}
	c := testChart(s)
	if p.price > 0 {
		c.Price = p.price
	}
	if p.currency != "" {
		c.Currency = p.currency
	}
	if p.date != "" {
		c.AsOf = p.date
	}
	return c, nil
}
func (p *fixtureProvider) News(context.Context, string) ([]Source, error) {
	if p.fail {
		return nil, errors.New("FIXTURE_NEWS_UNAVAILABLE")
	}
	return []Source{{Title: "TEST FIXTURE: NVDA earnings, not a real news item", URL: "https://example.com/news-fixture", Publisher: "TEST_FIXTURE", Kind: "news", Date: "2024-01-01", Text: "Synthetic browser acceptance input; not market evidence."}}, nil
}
func (p *fixtureProvider) Community(context.Context, string) ([]Source, error) {
	return []Source{{Title: "TEST FIXTURE: NVDA valuation discussion", URL: "https://example.com/opinion-fixture", Publisher: "TEST_FIXTURE", Kind: "opinion", Text: "This is an offline testing fixture, not a real post."}}, nil
}
func (p *fixtureProvider) Filings(context.Context, string, string) ([]Source, error) {
	return []Source{{Title: "TEST FIXTURE 10-K", URL: "https://example.com/filing-fixture", Publisher: "TEST_FIXTURE", Kind: "official"}}, nil
}
func (p *fixtureProvider) Facts(context.Context, string, string) ([]Fact, error) {
	return []Fact{{Label: "测试营业收入", Tag: "Revenues", Value: 123000000, Unit: "USD", Start: "2024-01-01", End: "2024-12-31", Filed: "2025-02-01", Form: "10-K", URL: "https://example.com/facts-fixture"}}, nil
}
func testApp(t *testing.T, p Provider) *App {
	t.Helper()
	s, e := openStore(filepath.Join(t.TempDir(), "data", "state.json"))
	if e != nil {
		t.Fatal(e)
	}
	return newApp(s, p)
}
func request(t *testing.T, a *App, method, path string, body any, cookie *http.Cookie) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var b io.Reader
	if body != nil {
		j, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		b = bytes.NewReader(j)
	}
	r := httptest.NewRequest(method, "http://127.0.0.1:18791"+path, b)
	r.Header.Set("X-InvestOne", "1")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	m := map[string]any{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w, m
}
func login(t *testing.T, a *App, key string) *http.Cookie {
	w, _ := request(t, a, "POST", "/api/login", map[string]string{"username": Username, "key": key}, nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	return w.Result().Cookies()[0]
}
func cfg() QuantConfig {
	return QuantConfig{Universe: "technology", Range: "2y", TopN: 3, FeeBPS: 10, Momentum: 50, LowVol: 30, Trend: 20}
}
func fixtures() map[string]Chart {
	m := map[string]Chart{}
	for _, s := range append(append([]string{}, technology...), "SPY") {
		m[s] = testChart(s)
	}
	return m
}
func near(t *testing.T, a, b float64) {
	t.Helper()
	if math.Abs(a-b) > 1e-8 {
		t.Fatalf("got %.12f want %.12f", a, b)
	}
}
func TestLoginUsername(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	for _, n := range []string{"", "admin", "红豆", "<script>"} {
		w, _ := request(t, a, "POST", "/api/login", map[string]string{"username": n, "key": ""}, nil)
		if w.Code != 401 {
			t.Fatal(n, w.Code)
		}
	}
	_ = login(t, a, "")
}
func TestSessionCookieSecurity(t *testing.T) {
	c := login(t, testApp(t, &fixtureProvider{}), "")
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || len(c.Value) < 32 {
		t.Fatal(c)
	}
}
func TestUnauthenticatedStateBlocked(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	w, _ := request(t, a, "GET", "/api/state", nil, nil)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestOriginAndHostBlocked(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	for _, tc := range []struct{ host, origin, site string }{{"attacker.example", "", ""}, {"127.0.0.1:18791", "https://evil.example", ""}, {"127.0.0.1:18791", "", "cross-site"}} {
		r := httptest.NewRequest("GET", "http://127.0.0.1:18791/api/health", nil)
		r.Host = tc.host
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Sec-Fetch-Site", tc.site)
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(tc, w.Code)
		}
	}
}
func TestMissingClientHeaderBlocked(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/api/login", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	a.handler().ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
func TestHealthAndStatic(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	w, m := request(t, a, "GET", "/api/health", nil, nil)
	if w.Code != 200 || m["mode"] != "LIVE_DATA_ONLY" {
		t.Fatal(m)
	}
	for _, p := range []string{"/", "/app.js", "/styles.css", "/favicon.svg"} {
		w, _ := request(t, a, "GET", p, nil, nil)
		if w.Code != 200 || w.Body.Len() < 50 {
			t.Fatal(p, w.Code)
		}
		if w.Header().Get("X-Frame-Options") != "DENY" {
			t.Fatal("missing frame protection")
		}
	}
}
func TestKeyNoPersistenceOrExport(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	key := "TEST_SECRET_NEVER_PERSIST_984512"
	c := login(t, a, key)
	p := a.store.snapshot().Profile
	p.Risk = "较低"
	w, _ := request(t, a, "POST", "/api/profile", p, c)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	b, e := os.ReadFile(a.store.path)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), key) {
		t.Fatal("key on disk")
	}
	for _, route := range []string{"/api/state", "/api/export"} {
		w, _ := request(t, a, "GET", route, nil, c)
		if strings.Contains(w.Body.String(), key) {
			t.Fatal("key in response")
		}
	}
	info, _ := os.Stat(a.store.path)
	if info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
}
func TestLogoutClearsKeyAndSession(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	c := login(t, a, "test-only")
	w, _ := request(t, a, "POST", "/api/logout", map[string]any{}, c)
	if w.Code != 200 || a.getKey() != "" {
		t.Fatal("logout failed")
	}
	w, _ = request(t, a, "GET", "/api/state", nil, c)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestNewlineKeyBlocked(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	w, _ := request(t, a, "POST", "/api/login", map[string]string{"username": Username, "key": "x\ny"}, nil)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestUnknownJSONFieldRejected(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	w, _ := request(t, a, "POST", "/api/login", map[string]string{"username": Username, "key": "", "password": "unexpected"}, nil)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestStateRestart(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	e := a.store.update(func(s *State) error { s.Watchlist = append(s.Watchlist, "TEST"); return nil })
	if e != nil {
		t.Fatal(e)
	}
	s, e := openStore(a.store.path)
	if e != nil || !reflect.DeepEqual(a.store.snapshot(), s.snapshot()) {
		t.Fatal("restart mismatch")
	}
}
func TestCorruptStoreNotOverwritten(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	_ = os.WriteFile(p, []byte("CORRUPT"), 0600)
	if _, e := openStore(p); e == nil {
		t.Fatal("accepted corrupt data")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "CORRUPT" {
		t.Fatal("overwritten")
	}
}
func TestUpdateRollback(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	orig := a.store.snapshot()
	_ = a.store.update(func(s *State) error { s.Cash = 0; return errors.New("rollback") })
	if !reflect.DeepEqual(orig, a.store.snapshot()) {
		t.Fatal("failed transaction changed state")
	}
}
func TestConcurrentStoreUpdates(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := a.store.update(func(s *State) error { s.Cash++; return nil }); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	near(t, a.store.snapshot().Cash, InitialCash+20)
}
func TestWeightedCostAndFee(t *testing.T) {
	s := initialState()
	if e := applyTrade(&s, "buy", "TEST", 2, 100, "FIXTURE", nowString()); e != nil {
		t.Fatal(e)
	}
	if e := applyTrade(&s, "buy", "TEST", 3, 200, "FIXTURE", nowString()); e != nil {
		t.Fatal(e)
	}
	near(t, s.Holdings[0].Cost, (200.1+600.3)/5)
	near(t, s.Cash, InitialCash-800.4)
}
func TestSellAndRealizedPnL(t *testing.T) {
	s := initialState()
	_ = applyTrade(&s, "buy", "TEST", 2, 100, "FIXTURE", nowString())
	_ = applyTrade(&s, "sell", "TEST", 1, 120, "FIXTURE", nowString())
	near(t, s.Holdings[0].Quantity, 1)
	near(t, s.Trades[1].Realized, 19.89)
	_ = applyTrade(&s, "sell", "TEST", 1, 120, "FIXTURE", nowString())
	if len(s.Holdings) != 0 {
		t.Fatal("zero holding remains")
	}
}
func TestInvalidTrades(t *testing.T) {
	for _, tc := range []struct {
		side string
		q, p float64
	}{{"sell", 1, 100}, {"buy", 1e9, 100}, {"buy", -1, 100}, {"buy", 1, -1}, {"buy", math.NaN(), 100}, {"buy", 1, math.Inf(1)}, {"other", 1, 100}} {
		s := initialState()
		before := clone(s)
		if e := applyTrade(&s, tc.side, "TEST", tc.q, tc.p, "FIXTURE", nowString()); e == nil {
			t.Fatal(tc)
		}
		if !reflect.DeepEqual(before, s) {
			t.Fatal("invalid trade changed state")
		}
	}
}
func TestNonUSDPaperTradeBlocked(t *testing.T) {
	a := testApp(t, &fixtureProvider{currency: "HKD"})
	c := login(t, a, "")
	w, _ := request(t, a, "POST", "/api/trade", map[string]any{"symbol": "0700.HK", "side": "buy", "quantity": 1}, c)
	if w.Code != 400 || a.store.snapshot().Cash != InitialCash {
		t.Fatal(w.Body.String())
	}
}
func TestStaleQuoteTradeBlocked(t *testing.T) {
	a := testApp(t, &fixtureProvider{date: "2001-01-01T00:00:00Z"})
	c := login(t, a, "")
	w, _ := request(t, a, "POST", "/api/trade", map[string]any{"symbol": "NVDA", "side": "buy", "quantity": 1}, c)
	if w.Code != 400 {
		t.Fatal(w.Body.String())
	}
}
func TestFailedProviderNoFakeQuotes(t *testing.T) {
	a := testApp(t, &fixtureProvider{fail: true})
	c := login(t, a, "")
	w, m := request(t, a, "GET", "/api/quotes?symbols=NVDA,MSFT", nil, c)
	if w.Code != 200 || len(m["quotes"].([]any)) != 0 || len(m["errors"].(map[string]any)) != 2 {
		t.Fatal(m)
	}
}
func TestYahooPreviousCloseNotRangeStart(t *testing.T) {
	b := []byte(`{"chart":{"result":[{"meta":{"regularMarketPrice":120,"regularMarketTime":1704326400,"chartPreviousClose":10,"currency":"USD"},"timestamp":[1704067200,1704153600,1704240000,1704326400],"indicators":{"quote":[{"close":[10,90,100,120]}],"adjclose":[{"adjclose":[8,80,95,118]}]}}],"error":null}}`)
	c, e := parseYahoo(b, "TEST", "")
	if e != nil {
		t.Fatal(e)
	}
	near(t, c.Previous, 100)
	near(t, c.Change, 20)
	near(t, c.Points[3].Close, 118)
	if !c.Adjusted {
		t.Fatal("adjustment missing")
	}
}
func TestYahooNullAndUnadjusted(t *testing.T) {
	b := []byte(`{"chart":{"result":[{"meta":{"regularMarketPrice":100,"regularMarketTime":1704326400,"currency":"USD"},"timestamp":[1704067200,1704153600,1704240000],"indicators":{"quote":[{"close":[98,null,100]}]}}]}}`)
	c, e := parseYahoo(b, "TEST", "")
	if e != nil || len(c.Points) != 2 || c.Adjusted {
		t.Fatal(c, e)
	}
}
func TestYahooErrors(t *testing.T) {
	for _, b := range []string{"bad json", `{"chart":{"error":{"description":"not found"}}}`, `{"chart":{"result":[]}}`} {
		if _, e := parseYahoo([]byte(b), "TEST", ""); e == nil {
			t.Fatal(b)
		}
	}
}
func TestFeedParseAndUnsafeLink(t *testing.T) {
	b := []byte(`<rss><channel><item><title>Good &amp; Test</title><link>https://example.com/a</link><description>&lt;b&gt;Data&lt;/b&gt;</description></item><item><title>bad</title><link>javascript:alert(1)</link></item></channel></rss>`)
	v, e := parseFeed(b, "news", "FIXTURE")
	if e != nil || len(v) != 1 || v[0].Title != "Good & Test" || v[0].Text != "Data" {
		t.Fatal(v, e)
	}
}
func TestAtomParse(t *testing.T) {
	b := []byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>FIXTURE post</title><link href="https://example.com/post"/><content>opinion</content><updated>2025-01-01</updated></entry></feed>`)
	v, e := parseFeed(b, "opinion", "FIXTURE")
	if e != nil || len(v) != 1 {
		t.Fatal(v, e)
	}
}
func TestSymbolValidation(t *testing.T) {
	for _, s := range []string{"NVDA", "0700.HK", "600519.SS", "000001.SZ", "BRK-B", "^GSPC"} {
		if _, e := symbol(s); e != nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"", "../etc/passwd", "AAPL?x=1", "https://example.com", "<script>"} {
		if _, e := symbol(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestQuantRunsAndFinite(t *testing.T) {
	r, e := backtest(fixtures(), technology, cfg())
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Curve) < 300 || len(r.Rebalances) < 10 {
		t.Fatal("incomplete result")
	}
	if !finite(r.Metrics.Sharpe) || !finite(r.Metrics.AnnualReturn) {
		t.Fatal(r.Metrics)
	}
	for _, x := range r.Rebalances {
		if x.SignalDate >= x.ExecutionDate {
			t.Fatal("no execution lag")
		}
	}
}
func TestQuantNoLookAhead(t *testing.T) {
	m := fixtures()
	base, e := backtest(m, technology, cfg())
	if e != nil {
		t.Fatal(e)
	}
	cut := 350
	cutdate := m[technology[0]].Points[cut].Date
	for s, c := range m {
		for i := cut; i < len(c.Points); i++ {
			c.Points[i].Close *= 1 + float64(i-cut+1)*float64(len(s))*.2
		}
		m[s] = c
	}
	changed, e := backtest(m, technology, cfg())
	if e != nil {
		t.Fatal(e)
	}
	for i, x := range base.Rebalances {
		if x.ExecutionDate < cutdate && !reflect.DeepEqual(x.Selections, changed.Rebalances[i].Selections) {
			t.Fatal("future prices changed past allocation")
		}
	}
	for i, x := range base.Curve {
		if x.Date < cutdate {
			near(t, x.Strategy, changed.Curve[i].Strategy)
		}
	}
}
func TestQuantMissingAssetRejected(t *testing.T) {
	m := fixtures()
	delete(m, "NVDA")
	if _, e := backtest(m, technology, cfg()); e == nil {
		t.Fatal("missing asset accepted")
	}
}
func TestQuantUnadjustedRejected(t *testing.T) {
	m := fixtures()
	c := m["NVDA"]
	c.Adjusted = false
	m["NVDA"] = c
	if _, e := backtest(m, technology, cfg()); e == nil {
		t.Fatal("unadjusted asset accepted")
	}
}
func TestQuantInsufficientHistoryRejected(t *testing.T) {
	m := fixtures()
	for s, c := range m {
		c.Points = c.Points[:150]
		m[s] = c
	}
	if _, e := backtest(m, technology, cfg()); e == nil {
		t.Fatal("insufficient history accepted")
	}
}
func TestQuantSingleAssetCost(t *testing.T) {
	m := map[string]Chart{"TEST": testChart("TEST"), "SPY": testChart("SPY")}
	for s, c := range m {
		for i := range c.Points {
			c.Points[i].Close = 100
		}
		m[s] = c
	}
	c := cfg()
	c.TopN = 1
	c.FeeBPS = 10
	r, e := backtest(m, []string{"TEST"}, c)
	if e != nil {
		t.Fatal(e)
	}
	near(t, r.Curve[len(r.Curve)-1].Strategy, .999)
	near(t, r.Metrics.Turnover, 1)
	near(t, r.Benchmark.TotalReturn, 0)
}
func TestQuantRiskOffCash(t *testing.T) {
	m := map[string]Chart{"TEST": testChart("TEST"), "SPY": testChart("SPY")}
	for s, c := range m {
		for i := range c.Points {
			c.Points[i].Close = 100 * math.Pow(.999, float64(i))
		}
		m[s] = c
	}
	c := cfg()
	c.TopN = 1
	c.RiskOff = true
	r, e := backtest(m, []string{"TEST"}, c)
	if e != nil {
		t.Fatal(e)
	}
	near(t, r.Curve[len(r.Curve)-1].Strategy, 1)
	near(t, r.Metrics.Turnover, 0)
}
func TestRankTies(t *testing.T) {
	if !reflect.DeepEqual(ranks([]float64{2, 1, 2}), []float64{1.5, 0, 1.5}) {
		t.Fatal("tie ranks")
	}
	if corr([]float64{1, 1, 1}, []float64{1, 2, 3}) != nil {
		t.Fatal("constant corr should be missing")
	}
}
func TestFactorInvalidWeights(t *testing.T) {
	c := cfg()
	c.Momentum = 0
	c.LowVol = 0
	c.Trend = 0
	if validateQuant(c, 8) == nil {
		t.Fatal("zero weights accepted")
	}
	c = cfg()
	c.FeeBPS = math.NaN()
	if validateQuant(c, 8) == nil {
		t.Fatal("nan accepted")
	}
}
func TestRuleDedupAndRearm(t *testing.T) {
	r := Rule{Type: "price_below", Threshold: 100}
	fire, latch := evaluatePrice(r, Chart{Price: 99})
	if !fire || !latch {
		t.Fatal("initial condition")
	}
	r.Latched = latch
	fire, latch = evaluatePrice(r, Chart{Price: 98})
	if fire || !latch {
		t.Fatal("duplicate")
	}
	_, r.Latched = evaluatePrice(r, Chart{Price: 101})
	fire, _ = evaluatePrice(r, Chart{Price: 99})
	if !fire {
		t.Fatal("not rearmed")
	}
}
func TestRuleCheckPersistsDedup(t *testing.T) {
	a := testApp(t, &fixtureProvider{price: 99})
	_ = a.store.update(func(s *State) error {
		s.Rules = append(s.Rules, Rule{ID: "rule1", Type: "price_below", Symbol: "NVDA", Threshold: 100, Enabled: true})
		return nil
	})
	if a.checkRules(context.Background()) != 1 || a.checkRules(context.Background()) != 0 || len(a.store.snapshot().Notices) != 1 {
		t.Fatal("rule dedup failed")
	}
}
func TestNewsRuleBaselineNoOldAlert(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	_ = a.store.update(func(s *State) error {
		s.Rules = append(s.Rules, Rule{ID: "rule1", Type: "keyword", Symbol: "NVDA", Keyword: "earnings", Enabled: true})
		return nil
	})
	if a.checkRules(context.Background()) != 0 || a.checkRules(context.Background()) != 0 {
		t.Fatal("old news alerted")
	}
	if a.store.snapshot().Rules[0].LastKey == "" {
		t.Fatal("baseline missing")
	}
}
func TestDocRetrieval(t *testing.T) {
	d := []Document{{Name: "NVDA research", Text: strings.Repeat("chips data center revenue increase. ", 100), Created: "2025-01-01"}, {Name: "other", Text: "wind rain unrelated"}}
	x := retrieveDocs(d, "NVDA revenue")
	if len(x) == 0 || !strings.Contains(x[0].Title, "NVDA") {
		t.Fatal(x)
	}
}
func TestNoKeyNoAI(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	_, e := a.chat(context.Background(), ChatRequest{Message: "hello", Symbol: "NVDA"}, true)
	if e == nil {
		t.Fatal("AI accepted missing key")
	}
}
func TestSharedPublicAILeavesNoChatHistory(t *testing.T) {
	t.Setenv("INVESTONE_CLOUD", "1")
	t.Setenv("INVESTONE_SHARED_AI_KEY", "fixture-public-key")
	a := testApp(t, &fixtureProvider{})
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Offline test reply [S1]."},"finish_reason":"stop"}]}`)
	}))
	defer gateway.Close()
	a.aiURL = gateway.URL
	res, err := a.chat(context.Background(), ChatRequest{Message:"Assess NVDA",Symbol:"NVDA",Persona:"assistant"}, true)
	if err != nil || res.Content == "" { t.Fatalf("public AI request: %+v %v", res, err) }
	if len(a.store.snapshot().Messages) != 0 { t.Fatal("public chat was retained in server state") }
	if saved, err := os.ReadFile(a.store.path); err == nil && strings.Contains(string(saved), "Offline test reply") {
		t.Fatal("public chat was written to disk")
	}
}
func TestAIRequestContractAndPrivacy(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	secret := "TEST_KEY_PROTOCOL_984613"
	login(t, a, secret)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("authorization")
		}
		if r.Header.Get("api-key") != secret {
			t.Error("ENT207 Azure APIM gateway requires api-key header")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != DefaultModel || body["max_tokens"] != float64(400) || body["temperature"] != .7 {
			t.Error("bad default payload", body["model"])
		}
		msgs := body["messages"].([]any)
		if !strings.Contains(msgs[0].(map[string]any)["content"].(string), "TEST_FIXTURE") {
			t.Error("missing evidence")
		}
		fmt.Fprintf(w, `{"choices":[{"message":{"content":"OFFLINE TEST RESPONSE [S1] %s"},"finish_reason":"stop"}],"usage":{"total_tokens":12}}`, secret)
	}))
	defer server.Close()
	a.aiURL = server.URL
	res, e := a.chat(context.Background(), ChatRequest{Message: "test NVDA", Symbol: "NVDA", Persona: "buffett"}, true)
	if e != nil {
		t.Fatal(e)
	}
	if calls != 1 || len(res.Sources) == 0 || strings.Contains(res.Content, secret) {
		t.Fatal(res)
	}
	b, _ := os.ReadFile(a.store.path)
	if strings.Contains(string(b), secret) {
		t.Fatal("key persisted in model echo")
	}
	if len(a.store.snapshot().Messages) != 2 {
		t.Fatal("history")
	}
}
func TestAIFailureNoRetry(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	login(t, a, "test")
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; http.Error(w, "upstream secret error", 429) }))
	defer s.Close()
	a.aiURL = s.URL
	_, e := a.chat(context.Background(), ChatRequest{Message: "test", Symbol: "NVDA"}, false)
	if e == nil || calls != 1 || strings.Contains(e.Error(), "upstream secret") {
		t.Fatal(e, calls)
	}
}
func TestAIEmptyResponseRejected(t *testing.T) {
	a := testApp(t, &fixtureProvider{})
	login(t, a, "test")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":""}}]}`)
	}))
	defer s.Close()
	a.aiURL = s.URL
	if _, e := a.chat(context.Background(), ChatRequest{Message: "test", Symbol: "NVDA"}, false); e == nil {
		t.Fatal("empty response accepted")
	}
}
func TestBrowserServer(t *testing.T) {
	if os.Getenv("INVESTONE_BROWSER_TEST") != "1" {
		t.Skip("browser harness only")
	}
	p := os.Getenv("INVESTONE_BROWSER_DATA")
	store, e := openStore(p)
	if e != nil {
		t.Fatal(e)
	}
	a := newApp(store, &fixtureProvider{})
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Offline test response; not real AI output. [S1]\n\nExamined the synthetic ticker evidence and paper portfolio context."},"finish_reason":"stop"}],"usage":{"total_tokens":17}}`)
	}))
	defer ai.Close()
	a.aiURL = ai.URL
	fmt.Println("BROWSER_FIXTURE_SERVER http://127.0.0.1:18792")
	if e := http.ListenAndServe("127.0.0.1:18792", a.handler()); e != nil {
		t.Fatal(e)
	}
}

func TestQuantExcludesIncompleteAndFutureBars(t *testing.T) {
	m := fixtures()
	base, err := backtest(m, technology, cfg())
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	for symbol, chart := range m {
		chart.Points = append(chart.Points, Point{Date: today, Close: 9999999}, Point{Date: "2099-01-01", Close: 1})
		m[symbol] = chart
	}
	got, err := backtest(m, technology, cfg())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base.Curve, got.Curve) || base.End != got.End {
		t.Fatal("incomplete bars altered backtest")
	}
}
func TestQuantMalformedDateRejected(t *testing.T) {
	m := fixtures()
	c := m["NVDA"]
	c.Points[0].Date = "bad"
	m["NVDA"] = c
	if _, err := backtest(m, technology, cfg()); err == nil {
		t.Fatal("malformed date accepted")
	}
}
