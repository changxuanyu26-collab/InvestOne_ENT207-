package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Stooq returns delayed daily bars, not a tradable realtime quote.
// It is an independent public fallback only for US equities and ETFs.
func (p *liveProvider) stooqMarket(ctx context.Context, s, rng string) (Chart, error) {
	if strings.Contains(s, ".") || strings.HasPrefix(s, "^") || strings.Contains(s, "=") {
		return Chart{}, errors.New("Stooq fallback supports US equity and ETF symbols only")
	}
	days := map[string]int{"1mo": 45, "3mo": 125, "6mo": 240, "1y": 410, "2y": 810, "5y": 1950}[rng]
	if days == 0 {
		days = 410
	}
	today := time.Now().UTC()
	from := today.AddDate(0, 0, -days)
	uri := "https://stooq.com/q/d/l/?s=" + url.QueryEscape(strings.ToLower(s)+".us") + "&i=d&d1=" + from.Format("20060102") + "&d2=" + today.Format("20060102")
	b, e := p.fetch(ctx, uri, "Mozilla/5.0 (compatible; InvestOne/1.2 personal-research)", 30*time.Minute)
	if e != nil {
		return Chart{}, e
	}
	rows, e := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if e != nil || len(rows) < 3 || len(rows[0]) < 5 || !strings.EqualFold(strings.TrimSpace(rows[0][0]), "Date") {
		return Chart{}, errors.New("Stooq did not return valid daily-price CSV")
	}
	c := Chart{Symbol: s, Name: s, Currency: "USD", Source: "Stooq · historical daily close (not real time; adjustment methodology unverified)", URL: "https://stooq.com/q/?s=" + url.QueryEscape(strings.ToLower(s)+".us"), Retrieved: nowString(), Points: []Point{}, Adjusted: false}
	for _, row := range rows[1:] {
		if len(row) < 5 {
			continue
		}
		d, e := time.Parse("2006-01-02", row[0])
		if e != nil || d.After(today.AddDate(0, 0, 1)) {
			continue
		}
		f, e := strconv.ParseFloat(row[4], 64)
		if e != nil || !finite(f) || f <= 0 {
			continue
		}
		c.Points = append(c.Points, Point{Date: d.Format("2006-01-02"), Close: f})
	}
	if len(c.Points) < 2 {
		return Chart{}, errors.New("Stooq returned fewer than two trading days")
	}
	// CSV is conventionally sorted ascending; reject unexpected format/order.
	for i := 1; i < len(c.Points); i++ {
		if c.Points[i].Date <= c.Points[i-1].Date {
			return Chart{}, errors.New("Stooq dates are not strictly increasing")
		}
	}
	last := c.Points[len(c.Points)-1]
	c.Price = last.Close
	c.Previous = c.Points[len(c.Points)-2].Close
	c.Change = (c.Price/c.Previous - 1) * 100
	c.AsOf = last.Date + "T00:00:00Z" // date only; not an exact exchange timestamp
	return c, nil
}

// Tencent's public unofficial China A-share day-K response. Historic close
// and latest daily close are always from one provider (no cross-source mix).
func (p *liveProvider) tencentMarket(ctx context.Context, s, rng string) (Chart, error) {
	suffix := ""
	currency := "CNY"
	if strings.HasSuffix(s, ".SS") {
		suffix = "sh" + strings.TrimSuffix(s, ".SS")
	}
	if strings.HasSuffix(s, ".SZ") {
		suffix = "sz" + strings.TrimSuffix(s, ".SZ")
	}
	if suffix == "" {
		return Chart{}, errors.New("Tencent daily-price fallback supports .SS/.SZ A-shares only")
	}
	n := map[string]int{"1mo": 35, "3mo": 90, "6mo": 160, "1y": 280, "2y": 550, "5y": 1250}[rng]
	if n == 0 {
		n = 280
	}
	uri := "https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=" + url.QueryEscape(fmt.Sprintf("%s,day,,,%d,qfq", suffix, n))
	b, e := p.fetch(ctx, uri, "Mozilla/5.0 (InvestOne/1.2; finance research)", 30*time.Minute)
	if e != nil {
		return Chart{}, e
	}
	// Some endpoints return an optional JSONP prefix; reject arbitrary scripts.
	trimmed := strings.TrimSpace(string(b))
	if !strings.HasPrefix(trimmed, "{") {
		i := strings.Index(trimmed, "=")
		if i < 0 || i > 70 {
			return Chart{}, errors.New("Tencent response format is unsupported")
		}
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed[i+1:], ";"))
	}
	var raw struct {
		Data map[string]map[string]json.RawMessage `json:"data"`
		Code int                                   `json:"code"`
	}
	if e := json.Unmarshal([]byte(trimmed), &raw); e != nil {
		return Chart{}, errors.New("Tencent daily-price response is not valid JSON")
	}
	node := raw.Data[suffix]
	vals := node["qfqday"]
	adjusted := len(vals) > 0 && string(vals) != "null"
	if !adjusted {
		vals = node["day"]
	}
	var rows [][]any
	if e := json.Unmarshal(vals, &rows); e != nil || len(rows) < 2 {
		return Chart{}, errors.New("Tencent returned no usable daily bars")
	}
	c := Chart{Symbol: s, Name: s, Currency: currency, Source: "Tencent Finance public daily-price endpoint (unofficial API)", URL: "https://gu.qq.com/", Retrieved: nowString(), Points: []Point{}, Adjusted: adjusted}
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		d := fmt.Sprint(row[0])
		if _, e := time.Parse("2006-01-02", d); e != nil {
			continue
		}
		v, e := strconv.ParseFloat(fmt.Sprint(row[2]), 64)
		if e != nil || !finite(v) || v <= 0 {
			continue
		}
		c.Points = append(c.Points, Point{Date: d, Close: v})
	}
	if len(c.Points) < 2 {
		return Chart{}, errors.New("Tencent returned fewer than two trading days")
	}
	for i := 1; i < len(c.Points); i++ {
		if c.Points[i].Date <= c.Points[i-1].Date {
			return Chart{}, errors.New("Tencent daily-price dates are not ordered")
		}
	}
	last := c.Points[len(c.Points)-1]
	c.Price = last.Close
	c.Previous = c.Points[len(c.Points)-2].Close
	c.Change = (c.Price/c.Previous - 1) * 100
	c.AsOf = last.Date + "T00:00:00Z" // daily date, not the precise trade time
	return c, nil
}
