package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (a *App) monitorLoop(ctx context.Context) {
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		a.checkRules(ctx)
	}
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.checkRules(ctx)
		}
	}
}
func evaluatePrice(r Rule, c Chart) (bool, bool) {
	condition := false
	switch r.Type {
	case "price_below":
		condition = c.Price <= r.Threshold
	case "price_above":
		condition = c.Price >= r.Threshold
	case "change_above":
		condition = c.Change >= r.Threshold
	case "change_below":
		condition = c.Change <= r.Threshold
	}
	return condition && !r.Latched, condition
}
func (a *App) checkRules(ctx context.Context) int {
	select {
	case a.monitorGate <- struct{}{}:
		defer func() { <-a.monitorGate }()
	default:
		return -1
	}
	st := a.store.snapshot()
	count := 0
	for _, r := range st.Rules {
		if !r.Enabled {
			continue
		}
		select {
		case <-ctx.Done():
			return count
		default:
		}
		trigger := false
		body := ""
		r.Error = ""
		r.LastCheck = nowString()
		switch r.Type {
		case "price_below", "price_above", "change_above", "change_below":
			c, e := a.provider.Market(ctx, r.Symbol, "1mo")
			if e != nil {
				r.Error = e.Error()
				break
			}
			asof, e := time.Parse(time.RFC3339, c.AsOf)
			if e != nil || time.Since(asof) > 5*24*time.Hour {
				r.Error = "Price timestamp is stale or missing; the rule will not trigger"
				break
			}
			if (r.Type == "change_above" || r.Type == "change_below") && c.Previous <= 0 {
				r.Error = "Previous close unavailable; the price-change rule will not trigger"
				break
			}
			trigger, r.Latched = evaluatePrice(r, c)
			body = fmt.Sprintf("%s reference price %.4f %s; change %.2f%%. Source %s, quote time %s.", r.Symbol, c.Price, c.Currency, c.Change, c.Source, c.AsOf)
		case "keyword", "filing":
			var items []Source
			var e error
			if r.Type == "filing" {
				items, e = a.provider.Filings(ctx, r.Symbol, st.Profile.Email)
			} else {
				items, e = a.provider.News(ctx, r.Symbol)
			}
			if e != nil {
				r.Error = e.Error()
				break
			}
			key := ""
			for _, i := range items {
				if r.Type == "filing" || strings.Contains(strings.ToLower(i.Title+" "+i.Text), strings.ToLower(r.Keyword)) {
					key = i.URL
					body = i.Title + "\n" + i.Date + "\n" + i.URL
					break
				}
			}
			if key != "" {
				trigger = r.LastKey != "" && key != r.LastKey
				r.LastKey = key
			}
		case "weekly":
			now := time.Now()
			y, w := now.ISOWeek()
			key := fmt.Sprintf("%d-W%02d", y, w)
			if now.Weekday() == time.Friday && now.Hour() >= 18 && r.LastKey != key {
				trigger = true
				r.LastKey = key
				body = fmt.Sprintf("Weekly paper portfolio review: cash $%.2f; %d holdings.", st.Cash, len(st.Holdings))
				for _, h := range st.Holdings {
					c, e := a.provider.Market(ctx, h.Symbol, "1mo")
					if e != nil {
						body += "\n" + h.Symbol + ": quote unavailable"
					} else {
						body += fmt.Sprintf("\n%s: %.4f shares; price %.2f %s; unrealized P&L %.2f USD; quote %s", h.Symbol, h.Quantity, c.Price, c.Currency, (c.Price-h.Cost)*h.Quantity, c.AsOf)
					}
				}
			}
		}
		if trigger {
			r.LastTriggered = nowString()
		}
		// Persist latch before a paid analysis so restarts do not duplicate the same charge/alert.
		n := Notice{ID: id(), RuleID: r.ID, Time: nowString(), Title: r.Symbol + " · automated monitoring alert", Body: body}
		found := false
		saved := a.store.update(func(s *State) error {
			for i := range s.Rules {
				if s.Rules[i].ID == r.ID && s.Rules[i].Enabled {
					s.Rules[i] = r
					found = true
					break
				}
			}
			if found && trigger {
				s.Notices = append(s.Notices, n)
				if len(s.Notices) > 500 {
					s.Notices = s.Notices[len(s.Notices)-500:]
				}
			}
			return nil
		})
		if saved != nil || !found {
			continue
		}
		if trigger {
			count++
			if r.AutoAI {
				result, e := a.chat(ctx, ChatRequest{Message: "This monitoring rule was triggered. Reassess using the currently available evidence: " + body, Symbol: r.Symbol, Persona: "assistant"}, false)
				extra := ""
				if e != nil {
					extra = "\n\nAutomated AI analysis failed: " + e.Error()
				} else {
					extra = "\n\nAI educational analysis:\n" + result.Content
					for _, s := range result.Sources {
						extra += "\n[" + s.ID + "] " + s.Title + " " + s.URL
					}
				}
				_ = a.store.update(func(s *State) error {
					for i := range s.Notices {
						if s.Notices[i].ID == n.ID {
							s.Notices[i].Body += extra
							break
						}
					}
					return nil
				})
			}
		}
	}
	return count
}
