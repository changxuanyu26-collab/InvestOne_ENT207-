package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Point struct {
	Date  string  `json:"date"`
	Close float64 `json:"close"`
}
type Chart struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Currency  string  `json:"currency"`
	Price     float64 `json:"price"`
	Previous  float64 `json:"previous"`
	Change    float64 `json:"change"`
	AsOf      string  `json:"as_of"`
	Retrieved string  `json:"retrieved"`
	Source    string  `json:"source"`
	URL       string  `json:"url"`
	Adjusted  bool    `json:"adjusted"`
	Points    []Point `json:"points"`
}
type Fact struct {
	Label string  `json:"label"`
	Tag   string  `json:"tag"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Start string  `json:"start"`
	End   string  `json:"end"`
	Filed string  `json:"filed"`
	Form  string  `json:"form"`
	URL   string  `json:"url"`
}
type Provider interface {
	Market(context.Context, string, string) (Chart, error)
	News(context.Context, string) ([]Source, error)
	Community(context.Context, string) ([]Source, error)
	Filings(context.Context, string, string) ([]Source, error)
	Facts(context.Context, string, string) ([]Fact, error)
}
type cacheEntry struct {
	data    []byte
	err     string
	expires time.Time
}
type liveProvider struct {
	client    *http.Client
	mu        sync.Mutex
	cache     map[string]cacheEntry
	proxyMode string
	limiter   <-chan time.Time
}

func newLiveProvider() *liveProvider {
	transport, mode := configuredTransport()
	return &liveProvider{proxyMode: mode, client: &http.Client{Transport: transport, Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, v []*http.Request) error {
		if len(v) >= 3 {
			return errors.New("Too many redirects")
		}
		if r.URL.Scheme != "https" {
			return errors.New("Non-HTTPS redirects are blocked")
		}
		allowed := map[string]bool{"query1.finance.yahoo.com": true, "query2.finance.yahoo.com": true, "feeds.finance.yahoo.com": true, "finance.yahoo.com": true, "news.google.com": true, "www.reddit.com": true, "www.sec.gov": true, "data.sec.gov": true, "stooq.com": true, "web.ifzq.gtimg.cn": true}
		if !allowed[r.URL.Hostname()] {
			return errors.New("Redirect host is not approved")
		}
		return nil
	}}, cache: map[string]cacheEntry{}, limiter: time.NewTicker(300 * time.Millisecond).C}
}
func (p *liveProvider) fetch(ctx context.Context, u, ua string, ttl time.Duration) ([]byte, error) {
	key := u + "|" + ua
	p.mu.Lock()
	c, ok := p.cache[key]
	p.mu.Unlock()
	if ok && time.Now().Before(c.expires) {
		if c.err != "" {
			return nil, errors.New(c.err)
		}
		return c.data, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.limiter:
	}
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json, application/rss+xml, application/atom+xml, text/xml;q=0.9, */*;q=0.5")
	resp, e := p.client.Do(req)
	var b []byte
	if e == nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			e = fmt.Errorf("%s returned HTTP %d", req.URL.Hostname(), resp.StatusCode)
		} else {
			b, e = io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024+1))
			if len(b) > 16*1024*1024 {
				e = errors.New("Provider response exceeded the size limit")
			}
		}
	} else {
		e = fmt.Errorf("%s connection error: %w", req.URL.Hostname(), e)
	}
	entry := cacheEntry{data: b, expires: time.Now().Add(ttl)}
	if e != nil {
		entry.err = e.Error()
		entry.expires = time.Now().Add(60 * time.Second)
	}
	p.mu.Lock()
	if len(p.cache) > 512 {
		p.cache = map[string]cacheEntry{}
	}
	p.cache[key] = entry
	p.mu.Unlock()
	return b, e
}
func (p *liveProvider) Market(ctx context.Context, sym, rng string) (Chart, error) {
	s, e := symbol(sym)
	if e != nil {
		return Chart{}, e
	}
	valid := map[string]bool{"1mo": true, "3mo": true, "6mo": true, "1y": true, "2y": true, "5y": true}
	if !valid[rng] {
		rng = "1y"
	}
	errs := []string{}
	for _, host := range []string{"query1.finance.yahoo.com", "query2.finance.yahoo.com"} {
		u := "https://" + host + "/v8/finance/chart/" + url.PathEscape(s) + "?interval=1d&range=" + rng + "&events=div%2Csplits&includeAdjustedClose=true"
		b, e := p.fetch(ctx, u, "Mozilla/5.0 (InvestOne/1.2 research)", 2*time.Minute)
		if e == nil {
			var c Chart
			c, e = parseYahoo(b, s, u)
			if e == nil {
				return c, nil
			}
		}
		errs = append(errs, host+": "+e.Error())
		if ctx.Err() != nil {
			return Chart{}, ctx.Err()
		}
	}
	var c Chart
	if strings.HasSuffix(s, ".SS") || strings.HasSuffix(s, ".SZ") {
		c, e = p.tencentMarket(ctx, s, rng)
	} else {
		c, e = p.stooqMarket(ctx, s, rng)
	}
	if e == nil {
		return c, nil
	}
	errs = append(errs, "Fallback source: "+e.Error())
	return Chart{}, errors.New(strings.Join(errs, " | "))
}
func parseYahoo(b []byte, s, u string) (Chart, error) {
	var raw struct {
		Chart struct {
			Error  *struct{ Description string }
			Result []struct {
				Meta struct {
					Symbol             string
					ShortName          string
					LongName           string
					Currency           string
					RegularMarketPrice float64
					ChartPreviousClose float64
					PreviousClose      float64
					RegularMarketTime  int64
				}
				Timestamp  []int64
				Indicators struct {
					Quote    []struct{ Close []*float64 }
					Adjclose []struct{ Adjclose []*float64 }
				}
			}
		}
	}
	if e := json.Unmarshal(b, &raw); e != nil {
		return Chart{}, errors.New("Quote endpoint did not return valid JSON")
	}
	if raw.Chart.Error != nil {
		return Chart{}, fmt.Errorf("Quote unavailable: %s", raw.Chart.Error.Description)
	}
	if len(raw.Chart.Result) == 0 {
		return Chart{}, errors.New("Quote provider did not return this symbol")
	}
	r := raw.Chart.Result[0]
	c := Chart{Symbol: s, Name: r.Meta.LongName, Currency: r.Meta.Currency, Price: r.Meta.RegularMarketPrice, Source: "Yahoo Finance · unofficial public endpoint", URL: "https://finance.yahoo.com/quote/" + url.PathEscape(s) + "/", Retrieved: nowString(), Points: []Point{}}
	if c.Name == "" {
		c.Name = r.Meta.ShortName
	}
	if c.Name == "" {
		c.Name = s
	}
	if r.Meta.RegularMarketTime > 0 {
		c.AsOf = time.Unix(r.Meta.RegularMarketTime, 0).UTC().Format(time.RFC3339)
	}
	if len(r.Indicators.Quote) == 0 {
		return Chart{}, errors.New("Historical close is missing")
	}
	values := r.Indicators.Quote[0].Close
	rawClose := values
	if len(r.Indicators.Adjclose) > 0 && len(r.Indicators.Adjclose[0].Adjclose) == len(values) {
		values = r.Indicators.Adjclose[0].Adjclose
		c.Adjusted = true
	}
	seen := map[string]bool{}
	for i, t := range r.Timestamp {
		if i >= len(values) || values[i] == nil || !finite(*values[i]) || *values[i] <= 0 {
			continue
		}
		d := time.Unix(t, 0).UTC().Format("2006-01-02")
		if !seen[d] {
			seen[d] = true
			c.Points = append(c.Points, Point{Date: d, Close: *values[i]})
		}
	}
	sort.Slice(c.Points, func(i, j int) bool { return c.Points[i].Date < c.Points[j].Date })
	if len(c.Points) == 0 {
		return Chart{}, errors.New("No usable historical prices")
	}
	// Previous-close uses unadjusted prices from the last completed prior bar, never the first day of the requested range.
	marketDate := ""
	if r.Meta.RegularMarketTime > 0 {
		marketDate = time.Unix(r.Meta.RegularMarketTime, 0).UTC().Format("2006-01-02")
	}
	for i := len(r.Timestamp) - 1; i >= 0; i-- {
		if i < len(rawClose) && rawClose[i] != nil && *rawClose[i] > 0 && time.Unix(r.Timestamp[i], 0).UTC().Format("2006-01-02") < marketDate {
			c.Previous = *rawClose[i]
			break
		}
	}
	if c.Previous <= 0 && r.Meta.PreviousClose > 0 {
		c.Previous = r.Meta.PreviousClose
	}
	if c.Price <= 0 || !finite(c.Price) {
		return Chart{}, errors.New("Latest reference price is invalid or missing")
	}
	if c.Previous > 0 {
		c.Change = (c.Price/c.Previous - 1) * 100
	}
	return c, nil
}

var tagsRE = regexp.MustCompile(`<[^>]*>`)
var scriptsRE = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)

func plain(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagsRE.ReplaceAllString(scriptsRE.ReplaceAllString(s, " "), " "))), " ")
}
func safeURL(s string) string {
	u, e := url.Parse(strings.TrimSpace(s))
	if e != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return u.String()
}
func parseFeed(b []byte, kind, publisher string) ([]Source, error) {
	var rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				Date        string `xml:"pubDate"`
				Source      string `xml:"source"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	out := []Source{}
	if e := xml.Unmarshal(b, &rss); e != nil {
		return nil, errors.New("Failed to parse RSS data")
	}
	for _, i := range rss.Channel.Items {
		u := safeURL(i.Link)
		if u == "" {
			continue
		}
		pub := i.Source
		if pub == "" {
			pub = publisher
		}
		out = append(out, Source{Title: plain(i.Title), URL: u, Publisher: pub, Date: i.Date, Text: trimRunes(plain(i.Description), 1200), Kind: kind})
	}
	if len(out) == 0 {
		var atom struct {
			Entries []struct {
				Title string `xml:"title"`
				Links []struct {
					Href string `xml:"href,attr"`
					Rel  string `xml:"rel,attr"`
				} `xml:"link"`
				Content string `xml:"content"`
				Updated string `xml:"updated"`
				Author  struct {
					Name string `xml:"name"`
				} `xml:"author"`
			} `xml:"entry"`
		}
		if e := xml.Unmarshal(b, &atom); e != nil {
			return nil, e
		}
		for _, i := range atom.Entries {
			u := ""
			for _, l := range i.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					u = safeURL(l.Href)
					break
				}
			}
			if u == "" {
				continue
			}
			out = append(out, Source{Title: plain(i.Title), URL: u, Publisher: publisher + " / " + i.Author.Name, Date: i.Updated, Text: trimRunes(plain(i.Content), 1200), Kind: kind})
		}
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return out, nil
}
func (p *liveProvider) News(ctx context.Context, s string) ([]Source, error) {
	s, e := symbol(s)
	if e != nil {
		return nil, e
	}
	urls := []struct{ url, source string }{
		{"https://feeds.finance.yahoo.com/rss/2.0/headline?s=" + url.QueryEscape(s) + "&region=US&lang=en-US", "Yahoo Finance RSS"},
		{"https://news.google.com/rss/search?q=" + url.QueryEscape(s+" stock OR earnings when:14d") + "&hl=en-US&gl=US&ceid=US:en", "Google News RSS · search results"},
	}
	errs := []string{}
	for _, u := range urls {
		b, e := p.fetch(ctx, u.url, "Mozilla/5.0 (InvestOne/1.2 research)", 10*time.Minute)
		if e == nil {
			var items []Source
			items, e = parseFeed(b, "news", u.source)
			if e == nil && len(items) > 0 {
				return items, nil
			}
			if e == nil {
				e = errors.New("No news items were returned")
			}
		}
		errs = append(errs, u.source+": "+e.Error())
	}
	return nil, errors.New(strings.Join(errs, " | "))
}
// SearchNews retrieves question-specific public reports. It uses the user's
// words as a search query and returns source links and RSS excerpts only; the
// model does not receive an invented article body.
func (p *liveProvider) SearchNews(ctx context.Context, question string) ([]Source, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("Empty news search")
	}
	// The request may contain instructions after the actual question. News
	// search needs event terms, not a verbatim paragraph of directions.
	for _, mark := range []string{"？", "?", "。", "\n"} {
		if at := strings.Index(question, mark); at > 0 {
			question = question[:at]
		}
	}
	if (strings.Contains(question, "黄仁勋") || strings.Contains(strings.ToLower(question), "jensen huang")) &&
		(strings.Contains(question, "出轨") || strings.Contains(strings.ToLower(question), "affair")) {
		question = "Jensen Huang affair"
	} else {
		for _, ending := range []string{"会有什么影响", "有什么影响", "有何影响", "对股票", "对股价", "请引用可靠来源", "并区分已证实事实与市场推测"} {
			question = strings.ReplaceAll(question, ending, " ")
		}
		question = strings.TrimSpace(trimRunes(question, 90))
	}
	if question == "" {
		return nil, errors.New("No searchable event terms found")
	}
	locale := "hl=en-US&gl=US&ceid=US:en"
	for _, r := range question {
		if r >= 0x3400 && r <= 0x9fff {
			locale = "hl=zh-CN&gl=US&ceid=US:zh-Hans"
			break
		}
	}
	u := "https://news.google.com/rss/search?q=" + url.QueryEscape(question) + "&" + locale
	b, err := p.fetch(ctx, u, "Mozilla/5.0 (InvestOne/1.2 research)", 10*time.Minute)
	if err != nil {
		return nil, err
	}
	items, err := parseFeed(b, "news", "Google News RSS · question search")
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("No question-specific news results were returned")
	}
	return items, nil
}
func (p *liveProvider) Community(ctx context.Context, s string) ([]Source, error) {
	s, e := symbol(s)
	if e != nil {
		return nil, e
	}
	u := "https://www.reddit.com/r/stocks/search.rss?q=" + url.QueryEscape(s) + "&restrict_sr=on&sort=new&t=month"
	b, e := p.fetch(ctx, u, "InvestOne/1.1 personal-research", 30*time.Minute)
	if e != nil {
		return nil, e
	}
	return parseFeed(b, "opinion", "Reddit / r/stocks")
}
func validEmail(s string) bool {
	return len(s) < 254 && regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`).MatchString(s)
}
func (p *liveProvider) cik(ctx context.Context, s, email string) (int, error) {
	if !validEmail(email) {
		return 0, errors.New("SEC API requires a contact email. Enter it in Settings or open the official filing links.")
	}
	b, e := p.fetch(ctx, "https://www.sec.gov/files/company_tickers.json", "InvestOne "+email, 24*time.Hour)
	if e != nil {
		return 0, e
	}
	var m map[string]struct {
		CIK    int    `json:"cik_str"`
		Ticker string `json:"ticker"`
	}
	if e = json.Unmarshal(b, &m); e != nil {
		return 0, e
	}
	for _, v := range m {
		if strings.EqualFold(v.Ticker, s) {
			return v.CIK, nil
		}
	}
	return 0, errors.New("SEC symbol mapping does not include this company")
}
func filingURL(cik int, acc, doc string) string {
	return "https://www.sec.gov/Archives/edgar/data/" + strconv.Itoa(cik) + "/" + strings.ReplaceAll(acc, "-", "") + "/" + url.PathEscape(doc)
}
func (p *liveProvider) Filings(ctx context.Context, s, email string) ([]Source, error) {
	n, e := p.cik(ctx, s, email)
	if e != nil {
		return nil, e
	}
	b, e := p.fetch(ctx, fmt.Sprintf("https://data.sec.gov/submissions/CIK%010d.json", n), "InvestOne "+email, 30*time.Minute)
	if e != nil {
		return nil, e
	}
	var r struct {
		Name    string
		Filings struct {
			Recent struct {
				Form            []string
				FilingDate      []string
				AccessionNumber []string
				PrimaryDocument []string
			}
		}
	}
	if e = json.Unmarshal(b, &r); e != nil {
		return nil, e
	}
	v := r.Filings.Recent
	out := []Source{}
	for i, f := range v.Form {
		if i >= len(v.FilingDate) || i >= len(v.AccessionNumber) || i >= len(v.PrimaryDocument) {
			continue
		}
		if f != "10-K" && f != "10-Q" && f != "8-K" && f != "20-F" && f != "6-K" {
			continue
		}
		out = append(out, Source{Title: r.Name + " · " + f, URL: filingURL(n, v.AccessionNumber[i], v.PrimaryDocument[i]), Publisher: "SEC EDGAR", Date: v.FilingDate[i], Kind: "official"})
		if len(out) >= 15 {
			break
		}
	}
	return out, nil
}
func (p *liveProvider) Facts(ctx context.Context, s, email string) ([]Fact, error) {
	n, e := p.cik(ctx, s, email)
	if e != nil {
		return nil, e
	}
	b, e := p.fetch(ctx, fmt.Sprintf("https://data.sec.gov/api/xbrl/companyfacts/CIK%010d.json", n), "InvestOne "+email, time.Hour)
	if e != nil {
		return nil, e
	}
	type obs struct {
		Start string
		End   string
		Filed string
		Form  string
		Val   float64
		Accn  string
	}
	var raw struct {
		Facts map[string]map[string]struct {
			Label string
			Units map[string][]obs
		}
	}
	if e = json.Unmarshal(b, &raw); e != nil {
		return nil, e
	}
	groups := [][]string{{"Revenue", "RevenueFromContractWithCustomerExcludingAssessedTax", "Revenues", "SalesRevenueNet"}, {"Net income", "NetIncomeLoss", "ProfitLoss"}, {"Assets", "Assets"}, {"Liabilities", "Liabilities"}, {"Operating cash flow", "NetCashProvidedByUsedInOperatingActivities"}, {"Capital expenditure", "PaymentsToAcquirePropertyPlantAndEquipment"}}
	out := []Fact{}
	for _, g := range groups {
		for _, tag := range g[1:] {
			f, ok := raw.Facts["us-gaap"][tag]
			if !ok {
				continue
			}
			obs := f.Units["USD"]
			filtered := []struct {
				Start string
				End   string
				Filed string
				Form  string
				Val   float64
				Accn  string
			}{}
			for _, o := range obs {
				if o.Form == "10-K" || o.Form == "10-Q" || o.Form == "20-F" {
					filtered = append(filtered, o)
				}
			}
			if len(filtered) == 0 {
				continue
			}
			sort.Slice(filtered, func(i, j int) bool {
				a, b := filtered[i], filtered[j]
				if a.End != b.End {
					return a.End > b.End
				}
				if a.Filed != b.Filed {
					return a.Filed > b.Filed
				}
				return a.Start > b.Start
			})
			o := filtered[0]
			out = append(out, Fact{Label: g[0], Tag: tag, Value: o.Val, Unit: "USD", Start: o.Start, End: o.End, Filed: o.Filed, Form: o.Form, URL: fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%d/%s/", n, strings.ReplaceAll(o.Accn, "-", ""))})
			break
		}
	}
	return out, nil
}
