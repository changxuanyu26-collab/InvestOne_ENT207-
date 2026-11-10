package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Visit counts are aggregate only. No IP address, user agent, question, or
// credential is stored in this file.
type visitCounts struct {
	PageViews       int64  `json:"page_views"`
	BrowserSessions int64  `json:"browser_sessions"`
	AIRequests      int64  `json:"ai_requests"`
	Day             string `json:"day"`
	AIRequestsToday int64  `json:"ai_requests_today"`
}

type visitTracker struct {
	mu   sync.Mutex
	path string
	data visitCounts
}

func newVisitTracker(root string) *visitTracker {
	v := &visitTracker{path: filepath.Join(root, "visits.json")}
	if b, err := os.ReadFile(v.path); err == nil {
		_ = json.Unmarshal(b, &v.data)
	}
	return v
}

func (v *visitTracker) save() {
	b, err := json.Marshal(v.data)
	if err != nil {
		return
	}
	tmp := v.path + ".tmp"
	if os.WriteFile(tmp, b, 0600) == nil {
		_ = os.Rename(tmp, v.path)
	}
}

func (v *visitTracker) pageView() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data.PageViews++
	v.save()
}

func (v *visitTracker) browserSession() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data.BrowserSessions++
	v.save()
}

func (v *visitTracker) allowAI() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	day := time.Now().UTC().Format("2006-01-02")
	if v.data.Day != day {
		v.data.Day = day
		v.data.AIRequestsToday = 0
	}
	// The public shared course quota is finite. Keep a bounded daily budget
	// even when visitors clear cookies or open private windows.
	if v.data.AIRequestsToday >= 250 {
		return false
	}
	v.data.AIRequestsToday++
	v.data.AIRequests++
	v.save()
	return true
}

func (v *visitTracker) snapshot() visitCounts {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.data
}
