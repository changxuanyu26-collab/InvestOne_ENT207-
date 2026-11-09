package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

var sectors = []string{"XLK", "XLY", "XLP", "XLE", "XLF", "XLV", "XLI", "XLB", "XLU", "XLRE", "XLC"}
var technology = []string{"AAPL", "MSFT", "NVDA", "GOOGL", "AMZN", "META", "AVGO", "AMD"}

type QuantConfig struct {
	Universe string  `json:"universe"`
	Range    string  `json:"range"`
	TopN     int     `json:"top_n"`
	FeeBPS   float64 `json:"fee_bps"`
	Momentum float64 `json:"momentum"`
	LowVol   float64 `json:"low_vol"`
	Trend    float64 `json:"trend"`
	RiskOff  bool    `json:"risk_off"`
}
type EquityPoint struct {
	Date      string  `json:"date"`
	Strategy  float64 `json:"strategy"`
	Benchmark float64 `json:"benchmark"`
	Drawdown  float64 `json:"drawdown"`
}
type Selection struct {
	Symbol string  `json:"symbol"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
}
type Rebalance struct {
	SignalDate    string      `json:"signal_date"`
	ExecutionDate string      `json:"execution_date"`
	Selections    []Selection `json:"selections"`
	Turnover      float64     `json:"turnover"`
	IC            *float64    `json:"ic"`
}
type Metrics struct {
	TotalReturn  float64  `json:"total_return"`
	AnnualReturn float64  `json:"annual_return"`
	Volatility   float64  `json:"volatility"`
	Sharpe       float64  `json:"sharpe"`
	MaxDrawdown  float64  `json:"max_drawdown"`
	Turnover     float64  `json:"turnover"`
	MeanIC       *float64 `json:"mean_ic"`
}
type QuantResult struct {
	Config       QuantConfig   `json:"config"`
	Metrics      Metrics       `json:"metrics"`
	Benchmark    Metrics       `json:"benchmark"`
	Curve        []EquityPoint `json:"curve"`
	Rebalances   []Rebalance   `json:"rebalances"`
	Sources      []Source      `json:"sources"`
	Warnings     []string      `json:"warnings"`
	Created      string        `json:"created"`
	Start        string        `json:"start"`
	End          string        `json:"end"`
	Observations int           `json:"observations"`
}

func validateQuant(c QuantConfig, n int) error {
	if c.TopN < 1 || c.TopN > n {
		return errors.New("Top N is larger than the available universe")
	}
	if !finite(c.FeeBPS) || c.FeeBPS < 0 || c.FeeBPS > 100 {
		return errors.New("Fee must be between 0 and 100 bps")
	}
	sum := 0.
	for _, x := range []float64{c.Momentum, c.LowVol, c.Trend} {
		if !finite(x) || x < 0 || x > 100 {
			return errors.New("Factor weights must be between 0 and 100")
		}
		sum += x
	}
	if sum <= 0 {
		return errors.New("At least one factor weight must be positive")
	}
	return nil
}
func mean(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	s := 0.
	for _, v := range x {
		s += v
	}
	return s / float64(len(x))
}
func std(x []float64) float64 {
	if len(x) < 2 {
		return 0
	}
	m := mean(x)
	s := 0.
	for _, v := range x {
		s += (v - m) * (v - m)
	}
	return math.Sqrt(s / float64(len(x)-1))
}
func zscore(x []float64) []float64 {
	r := make([]float64, len(x))
	m, v := mean(x), std(x)
	if v < 1e-14 {
		return r
	}
	for i, a := range x {
		r[i] = (a - m) / v
	}
	return r
}
func ranks(x []float64) []float64 {
	idx := make([]int, len(x))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool { return x[idx[i]] < x[idx[j]] })
	r := make([]float64, len(x))
	for a := 0; a < len(idx); {
		b := a + 1
		for b < len(idx) && x[idx[b]] == x[idx[a]] {
			b++
		}
		rank := float64(a+b-1) / 2
		for k := a; k < b; k++ {
			r[idx[k]] = rank
		}
		a = b
	}
	return r
}
func corr(x, y []float64) *float64 {
	if len(x) != len(y) || len(x) < 3 {
		return nil
	}
	mx, my := mean(x), mean(y)
	a, b, c := 0., 0., 0.
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		a += dx * dy
		b += dx * dx
		c += dy * dy
	}
	if b*c < 1e-20 {
		return nil
	}
	r := a / math.Sqrt(b*c)
	return &r
}
func factorScores(prices [][]float64, k int, c QuantConfig) ([]float64, []bool) {
	n := len(prices)
	m, v, t := make([]float64, n), make([]float64, n), make([]float64, n)
	eligible := make([]bool, n)
	for j, p := range prices {
		m[j] = 0.5*(p[k]/p[k-63]-1) + 0.5*(p[k]/p[k-126]-1)
		rets := []float64{}
		for d := k - 62; d <= k; d++ {
			rets = append(rets, p[d]/p[d-1]-1)
		}
		v[j] = -std(rets)
		t[j] = p[k]/mean(p[k-125:k+1]) - 1
		eligible[j] = !c.RiskOff || (m[j] > 0 && t[j] > 0)
	}
	mz, vz, tz := zscore(m), zscore(v), zscore(t)
	score := make([]float64, n)
	den := c.Momentum + c.LowVol + c.Trend
	for j := range score {
		score[j] = (c.Momentum*mz[j] + c.LowVol*vz[j] + c.Trend*tz[j]) / den
	}
	return score, eligible
}
func calculateMetrics(vals, returns []float64) Metrics {
	m := Metrics{}
	if len(vals) < 2 {
		return m
	}
	m.TotalReturn = vals[len(vals)-1]/vals[0] - 1
	m.AnnualReturn = math.Pow(1+m.TotalReturn, 252/float64(len(returns))) - 1
	m.Volatility = std(returns) * math.Sqrt(252)
	if std(returns) > 1e-14 {
		m.Sharpe = mean(returns) / std(returns) * math.Sqrt(252)
	}
	peak := vals[0]
	for _, v := range vals {
		if v > peak {
			peak = v
		}
		m.MaxDrawdown = math.Max(m.MaxDrawdown, 1-v/peak)
	}
	return m
}
func backtest(charts map[string]Chart, syms []string, c QuantConfig) (QuantResult, error) {
	if e := validateQuant(c, len(syms)); e != nil {
		return QuantResult{}, e
	}
	all := append(append([]string{}, syms...), "SPY")
	maps := map[string]map[string]float64{}
	dateCount := map[string]int{}
	seenSymbols := map[string]bool{}
	// Conservative complete-day boundary: exclude today even after the US close.
	completeBefore := time.Now().UTC().Format("2006-01-02")
	for _, s := range all {
		if seenSymbols[s] {
			continue
		}
		seenSymbols[s] = true
		ch, ok := charts[s]
		if !ok || !ch.Adjusted {
			return QuantResult{}, fmt.Errorf("%s lacks verified adjusted prices; declining to run an inconsistent backtest", s)
		}
		if ch.Currency != "USD" {
			return QuantResult{}, errors.New("This backtest requires a fully USD-denominated universe")
		}
		m := map[string]float64{}
		for _, p := range ch.Points {
			if _, err := time.Parse("2006-01-02", p.Date); err != nil {
				return QuantResult{}, fmt.Errorf("%s has an invalid historical date format", s)
			}
			if p.Date < completeBefore && p.Close > 0 && finite(p.Close) {
				m[p.Date] = p.Close
			}
		}
		maps[s] = m
		for d := range m {
			dateCount[d]++
		}
	}
	dates := []string{}
	for d, n := range dateCount {
		if n == len(seenSymbols) {
			dates = append(dates, d)
		}
	}
	sort.Strings(dates)
	if len(dates) < 180 {
		return QuantResult{}, errors.New("Fewer than 180 common trading days; insufficient for warmup and backtest. Missing prices are not filled.")
	}
	prices := make([][]float64, len(syms))
	for j, s := range syms {
		for _, d := range dates {
			prices[j] = append(prices[j], maps[s][d])
		}
	}
	bench := []float64{}
	for _, d := range dates {
		bench = append(bench, maps["SPY"][d])
	}
	start := 127
	nav := 1.
	bnav := 1.
	peak := 1.
	w := make([]float64, len(syms))
	cash := 1.
	prevMonth := ""
	totalTurnover := 0.
	strategyValues := []float64{1}
	benchValues := []float64{1}
	rets, brets, ics := []float64{}, []float64{}, []float64{}
	result := QuantResult{Config: c, Created: nowString(), Start: dates[start], End: dates[len(dates)-1], Curve: []EquityPoint{{Date: dates[start], Strategy: 1, Benchmark: 1, Drawdown: 0}}, Rebalances: []Rebalance{}, Sources: []Source{}, Warnings: []string{"Historical educational simulation, not a return forecast. A fixed universe introduces selection and survivorship bias.", "Only common valid trading days are used. Adjusted prices reflect the current provider version, not a point-in-time database.", "Current UTC date and future daily bars are excluded to avoid treating incomplete sessions as finished.", "Signals use previous-day close; approximate execution occurs at next close; returns begin afterward.", "Fees are charged on gross dollar turnover; market impact, taxes and execution constraints are excluded.", "Sharpe assumes a zero risk-free rate. IC is forward 21-trading-day cross-sectional rank IC, reported ex post for diagnostics only."}}
	// On iteration i, signal uses close[i-2], execution close[i-1], and only return i-1 -> i is earned.
	for i := start + 1; i < len(dates); i++ {
		feeFactor := 1.
		month := dates[i-1][:7]
		if month != prevMonth {
			prevMonth = month
			score, eligible := factorScores(prices, i-2, c)
			idx := []int{}
			for j := range syms {
				if eligible[j] {
					idx = append(idx, j)
				}
			}
			sort.SliceStable(idx, func(a, b int) bool {
				if score[idx[a]] == score[idx[b]] {
					return syms[idx[a]] < syms[idx[b]]
				}
				return score[idx[a]] > score[idx[b]]
			})
			if len(idx) > c.TopN {
				idx = idx[:c.TopN]
			}
			target := make([]float64, len(w))
			for _, j := range idx {
				target[j] = 1 / float64(c.TopN)
			}
			turn := 0.
			sels := []Selection{}
			for j := range w {
				turn += math.Abs(target[j] - w[j])
				if target[j] > 0 {
					sels = append(sels, Selection{Symbol: syms[j], Weight: target[j], Score: score[j]})
				}
			}
			feeFactor = 1 - turn*c.FeeBPS/10000
			totalTurnover += turn
			w = target
			cash = 1.
			for _, x := range w {
				cash -= x
			}
			r := Rebalance{SignalDate: dates[i-2], ExecutionDate: dates[i-1], Selections: sels, Turnover: turn}
			if i-1+21 < len(dates) {
				forward := make([]float64, len(syms))
				for j := range syms {
					forward[j] = prices[j][i-1+21]/prices[j][i-1] - 1
				}
				r.IC = corr(ranks(score), ranks(forward))
				if r.IC != nil {
					ics = append(ics, *r.IC)
				}
			}
			result.Rebalances = append(result.Rebalances, r)
		}
		growth := cash
		for j := range w {
			growth += w[j] * prices[j][i] / prices[j][i-1]
		}
		net := feeFactor*growth - 1
		nav *= 1 + net
		br := bench[i]/bench[i-1] - 1
		bnav *= 1 + br
		rets = append(rets, net)
		brets = append(brets, br)
		strategyValues = append(strategyValues, nav)
		benchValues = append(benchValues, bnav)
		if nav > peak {
			peak = nav
		}
		result.Curve = append(result.Curve, EquityPoint{Date: dates[i], Strategy: nav, Benchmark: bnav, Drawdown: nav/peak - 1})
		for j := range w {
			w[j] = w[j] * (prices[j][i] / prices[j][i-1]) / growth
		}
		cash /= growth
	}
	result.Metrics = calculateMetrics(strategyValues, rets)
	result.Metrics.Turnover = totalTurnover
	if len(ics) > 0 {
		v := mean(ics)
		result.Metrics.MeanIC = &v
	}
	result.Benchmark = calculateMetrics(benchValues, brets)
	result.Observations = len(rets)
	for _, s := range all {
		ch := charts[s]
		result.Sources = append(result.Sources, Source{Title: s + " adjusted daily close", URL: ch.URL, Publisher: ch.Source, Date: ch.Retrieved, Kind: "market"})
	}
	return result, nil
}
func (a *App) runQuant(ctx context.Context, c QuantConfig) (QuantResult, error) {
	syms := sectors
	if c.Universe == "technology" {
		syms = technology
	} else if c.Universe != "sectors" {
		return QuantResult{}, errors.New("Choose a sector ETF or technology-equity universe")
	}
	if c.Range != "2y" && c.Range != "5y" {
		c.Range = "2y"
	}
	if e := validateQuant(c, len(syms)); e != nil {
		return QuantResult{}, e
	}
	select {
	case a.quantGate <- struct{}{}:
		defer func() { <-a.quantGate }()
	default:
		return QuantResult{}, errors.New("A backtest is already running")
	}
	charts := map[string]Chart{}
	var mu sync.Mutex
	errs := []string{}
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for _, s := range append(append([]string{}, syms...), "SPY") {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ch, e := a.provider.Market(ctx, s, c.Range)
			mu.Lock()
			defer mu.Unlock()
			if e != nil {
				errs = append(errs, s+": "+e.Error())
			} else {
				charts[s] = ch
			}
		}(s)
	}
	wg.Wait()
	if len(errs) > 0 {
		sort.Strings(errs)
		return QuantResult{}, fmt.Errorf("Real historical data are incomplete; no backtest was run. %s", errs[0])
	}
	return backtest(charts, syms, c)
}
