package main

import (
	"context"
	"strings"
	"testing"
)

func TestParseHackerNewsCommunity(t *testing.T) {
	b := []byte(`{"hits":[{"author":"researcher","created_at":"2026-11-18T12:30:00Z","num_comments":42,"objectID":"12345","points":88,"title":"Nvidia growth and valuation discussion","url":"https://example.com/evidence"}]}`)
	items, err := parseHackerNewsCommunity(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items", len(items))
	}
	item := items[0]
	if item.Kind != "opinion" || item.URL != "https://news.ycombinator.com/item?id=12345" {
		t.Fatalf("unexpected community item: %+v", item)
	}
	if !strings.Contains(item.Publisher, "88 points") || !strings.Contains(item.Publisher, "42 comments") {
		t.Fatalf("engagement metadata missing: %q", item.Publisher)
	}
}

func TestCommunitySearchTermsUseCompanyNameAndTicker(t *testing.T) {
	terms := communitySearchTerms("NVDA")
	if len(terms) != 2 || terms[0] != "Nvidia" || terms[1] != "NVDA" {
		t.Fatalf("unexpected terms: %#v", terms)
	}
}

func TestCommunityTitleRelevance(t *testing.T) {
	if !communityTitleMatches("NVDA", "Nvidia raises its guidance") {
		t.Fatal("company-name title was rejected")
	}
	if !communityTitleMatches("NVDA", "NVDA valuation discussion") {
		t.Fatal("ticker title was rejected")
	}
	if communityTitleMatches("NVDA", "Microsoft ONNX Runtime has disappeared from GitHub") {
		t.Fatal("unrelated title was accepted because an author or body matched")
	}
}

type communityErrorProvider struct{ fixtureProvider }

func (p *communityErrorProvider) Community(context.Context, string) ([]Source, error) {
	return nil, errCommunityTestUpstream
}

var errCommunityTestUpstream = &communityTestError{}

type communityTestError struct{}

func (*communityTestError) Error() string { return "www.reddit.com returned HTTP 404" }

func TestCommunityEndpointHidesRawUpstreamErrors(t *testing.T) {
	a := testApp(t, &communityErrorProvider{})
	cookie := login(t, a, "")
	w, body := request(t, a, "GET", "/api/community?symbol=NVDA", nil, cookie)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	message, _ := body["error"].(string)
	if strings.Contains(message, "404") || strings.Contains(strings.ToLower(message), "reddit") {
		t.Fatalf("raw upstream error leaked: %q", message)
	}
	if !strings.Contains(message, "temporarily unavailable") {
		t.Fatalf("friendly fallback missing: %q", message)
	}
}
