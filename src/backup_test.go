package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func backupFixtureProvider(f roundTrip) *liveProvider {
	p := newLiveProvider()
	p.client = &http.Client{Transport: f, Timeout: time.Second}
	return p
}
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestFallbackStooqNoFakeData(t *testing.T) {
	seen := []string{}
	p := backupFixtureProvider(func(r *http.Request) (*http.Response, error) {
		seen = append(seen, r.URL.Hostname())
		if r.URL.Hostname() == "stooq.com" {
			return response(200, "Date,Open,High,Low,Close,Volume\n2026-10-07,100,101,99,100,1100\n2026-10-08,101,104,99,103,1800\n2026-10-09,104,105,100,104,2000\n"), nil
		}
		return response(429, "limit"), nil
	})
	c, e := p.Market(context.Background(), "NVDA", "1mo")
	if e != nil {
		t.Fatal(e)
	}
	if c.Source == "" || c.Price != 104 || c.Previous != 103 || c.Adjusted || len(c.Points) != 3 || !strings.Contains(c.Source, "Stooq") {
		t.Fatalf("unexpected fallback: %+v", c)
	}
	if strings.Join(seen, ",") != "query1.finance.yahoo.com,query2.finance.yahoo.com,stooq.com" {
		t.Fatalf("unexpected order: %v", seen)
	}
}
func TestFallbackTencentA(t *testing.T) {
	p := backupFixtureProvider(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "web.ifzq.gtimg.cn" {
			return response(200, `{"data":{"sh600519":{"qfqday":[["2026-10-08","1500","1505","1510","1490","99"],["2026-10-09","1506","1520","1522","1500","100"]]}}}`), nil
		}
		return nil, errors.New("unreachable")
	})
	c, e := p.Market(context.Background(), "600519.SS", "1mo")
	if e != nil || c.Price != 1520 || c.Previous != 1505 || c.Currency != "CNY" || !c.Adjusted {
		t.Fatalf("bad tencent: %+v %v", c, e)
	}
}
func TestFallbackGoogleNews(t *testing.T) {
	p := backupFixtureProvider(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "news.google.com" {
			return response(200, `<rss version="2.0"><channel><item><title>NVDA reports earnings</title><link>https://news.google.com/articles/test</link><pubDate>Fri, 09 Oct 2026 09:00:00 GMT</pubDate></item></channel></rss>`), nil
		}
		return response(404, "not found"), nil
	})
	items, e := p.News(context.Background(), "NVDA")
	if e != nil || len(items) != 1 || items[0].Title != "NVDA reports earnings" {
		t.Fatalf("bad rss: %+v %v", items, e)
	}
}
func TestQuestionSpecificNewsSearch(t *testing.T) {
	p := backupFixtureProvider(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() != "news.google.com" || r.URL.Query().Get("q") != "黄仁勋出轨对股票有什么影响" || r.URL.Query().Get("hl") != "zh-CN" {
			t.Fatalf("unexpected question search: %s", r.URL.String())
		}
		return response(200, `<rss version="2.0"><channel><item><title>NVIDIA publishes a company statement</title><link>https://news.google.com/articles/statement</link><pubDate>Mon, 09 Nov 2026 09:00:00 GMT</pubDate></item></channel></rss>`), nil
	})
	items, err := p.SearchNews(context.Background(), "黄仁勋出轨对股票有什么影响")
	if err != nil || len(items) != 1 || items[0].Publisher == "" {
		t.Fatalf("question-specific search failed: %+v %v", items, err)
	}
}
func TestAllFailedNeverSynthetic(t *testing.T) {
	p := backupFixtureProvider(func(r *http.Request) (*http.Response, error) { return nil, errors.New("dial test unavailable") })
	c, e := p.Market(context.Background(), "NVDA", "1mo")
	if e == nil || c.Price != 0 || !strings.Contains(e.Error(), "stooq.com") {
		t.Fatalf("must fail explicitly: %+v %v", c, e)
	}
}
func TestMacManualProxyParsing(t *testing.T) {
	conf := parseSystemProxy("<dictionary> {\n HTTPSEnable : 1\n HTTPSProxy : 127.0.0.1\n HTTPSPort : 7890\n}")
	if conf.proxy == nil || conf.proxy.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy: %+v", conf)
	}
	pac := parseSystemProxy("ProxyAutoConfigEnable : 1\nProxyAutoConfigURLString : https://example.com/proxy.pac")
	if pac.proxy != nil || !strings.Contains(pac.Mode, "PAC") {
		t.Fatalf("PAC must not be falsely supported: %+v", pac)
	}
}
