package main

import (
    "context"
    "os"
    "strings"
    "testing"
    "time"
)

// TestLiveFinancialProviders performs real provider requests, not mocked responses.
// Invoke explicitly: INVESTONE_LIVE_PROBE=1 go test -v -run TestLiveFinancialProviders.
// Outside that environment it is skipped, making offline unit tests deterministic.
func TestLiveFinancialProviders(t *testing.T) {
    if os.Getenv("INVESTONE_LIVE_PROBE") != "1" {
        t.Skip("Explicitly opt in to external financial-data requests")
    }
    provider := newLiveProvider()
    for _, tc := range []struct{ symbol, window string }{
        {"SPY","1mo"}, {"NVDA","1mo"}, {"600519.SS","1mo"},
    } {
        tc := tc
        t.Run("market_"+strings.ReplaceAll(tc.symbol,".","_"),func(t *testing.T) {
            ctx,cancel:=context.WithTimeout(context.Background(),50*time.Second)
            defer cancel()
            quote,err:=provider.Market(ctx,tc.symbol,tc.window)
            if err!=nil { t.Fatalf("REAL PROVIDER FAILED: %s: %v",tc.symbol,err) }
            if quote.Symbol!=tc.symbol || quote.Price<=0 || quote.Previous<=0 || len(quote.Points)<2 || quote.Source=="" || quote.URL=="" || quote.AsOf=="" || quote.Retrieved=="" {
                t.Fatalf("Malformed/stale provenance or incomplete market response: %+v",quote)
            }
            for i:=1;i<len(quote.Points);i++ {
                if quote.Points[i].Date<=quote.Points[i-1].Date { t.Fatalf("Non-increasing price dates for %s",tc.symbol) }
            }
            if _,err=time.Parse(time.RFC3339,quote.AsOf);err!=nil{t.Fatalf("Invalid quote timestamp: %v",err)}
            t.Logf("REAL_MARKET_PASS symbol=%s provider=%q quote=%f %s quote_asof=%s retrieved=%s bars=%d adjusted=%t",
                quote.Symbol,quote.Source,quote.Price,quote.Currency,quote.AsOf,quote.Retrieved,len(quote.Points),quote.Adjusted)
        })
    }
    t.Run("financial_news_NVDA",func(t *testing.T){
        ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second)
        defer cancel()
        items,err:=provider.News(ctx,"NVDA")
        if err!=nil{t.Fatalf("REAL NEWS PROVIDER FAILED: %v",err)}
        if len(items)==0 || items[0].Publisher=="" || items[0].URL=="" || items[0].Date==""{t.Fatalf("News source or date missing: %+v",items)}
        t.Logf("REAL_NEWS_PASS items=%d source=%q date=%q url=%q",len(items),items[0].Publisher,items[0].Date,items[0].URL)
    })
    t.Run("public_forum",func(t *testing.T){
        ctx,cancel:=context.WithTimeout(context.Background(),25*time.Second)
        defer cancel()
        items,err:=provider.Community(ctx,"NVDA")
        if err!=nil{
            t.Logf("REAL_COMMUNITY_UNAVAILABLE: %v (forums frequently restrict access; users can still add sourced notes)",err)
            return
        }
        t.Logf("REAL_COMMUNITY_PASS items=%d",len(items))
    })
    if contact:=os.Getenv("SEC_CONTACT_EMAIL");contact!=""{
       t.Run("SEC_company_facts",func(t *testing.T){
           ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second)
           defer cancel()
           facts,err:=provider.Facts(ctx,"NVDA",contact)
           if err!=nil {t.Fatalf("REAL SEC PROVIDER FAILED: %v",err)}
           if len(facts)==0 {t.Fatal("SEC returned no financial facts")}
           t.Logf("REAL_SEC_PASS facts=%d filing_date=%s tag=%s",len(facts),facts[0].Filed,facts[0].Tag)
       })
    } else {
       t.Log("SEC connectivity not exercised: SEC_CONTACT_EMAIL must be supplied by a legitimate contact")
    }
}
