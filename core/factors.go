// Package core contains portable educational quantitative building blocks.
// It is a small independently testable extract of InvestOne's factor research
// methodology; the full desktop application additionally contains historical
// feed adapters, monthly rebalancing, metrics, and HTTP handlers.
package core

import (
 "errors"
 "math"
 "sort"
)

type Observation struct {
 Symbol string
 Momentum float64 // point-in-time lookback return
 Volatility float64 // point-in-time realized volatility
 Trend float64 // point-in-time trend score
}

type FactorWeights struct {
 Momentum float64
 LowVolatility float64
 Trend float64
}

type RankedAsset struct {
 Symbol string
 Score float64
 Weight float64
}

// Select ranks assets using cross-sectional z-scores and selects up to topN.
// All observations must have been computed strictly from dates preceding the
// modeled execution date. It returns normalized equal weights for selected
// candidates and does not itself execute trades or forecast future returns.
func Select(xs []Observation, w FactorWeights, topN int, riskOff bool) ([]RankedAsset,error) {
 n:=len(xs)
 if n==0 || topN<1 || topN>n { return nil,errors.New("invalid universe or selection count") }
 weights:=[]float64{w.Momentum,w.LowVolatility,w.Trend}
 total:=0.0
 for _,v:=range weights {
  if math.IsNaN(v)||math.IsInf(v,0)||v<0 {return nil,errors.New("factor weights must be finite and nonnegative")}
  total+=v
 }
 if total<=0 {return nil,errors.New("at least one factor weight must be positive")}
 names:=map[string]bool{}
 data:=make([][]float64,3)
 for _,x:=range xs {
  if x.Symbol==""||names[x.Symbol] {return nil,errors.New("missing or duplicate ticker")}
  names[x.Symbol]=true
  vs:=[]float64{x.Momentum,-x.Volatility,x.Trend}
  for j,v:=range vs {if math.IsNaN(v)||math.IsInf(v,0) {return nil,errors.New("nonfinite factor input")};data[j]=append(data[j],v)}
 }
 zscores:=make([][]float64,3)
 for j,col:=range data {
  mean:=0.0;for _,v:=range col{mean+=v};mean/=float64(n)
  variance:=0.0;for _,v:=range col{variance+=(v-mean)*(v-mean)}
  variance/=float64(n)
  zscores[j]=make([]float64,n)
  if variance>1e-20 {
   sd:=math.Sqrt(variance)
   for i,v:=range col{zscores[j][i]=(v-mean)/sd}
  }
 }
 ranked:=make([]RankedAsset,0,n)
 for i,x:=range xs {
  if riskOff && (x.Momentum<=0 || x.Trend<=0) {continue}
  score:=(w.Momentum*zscores[0][i]+w.LowVolatility*zscores[1][i]+w.Trend*zscores[2][i])/total
  ranked=append(ranked,RankedAsset{Symbol:x.Symbol,Score:score})
 }
 sort.Slice(ranked,func(i,j int)bool {
  if ranked[i].Score==ranked[j].Score{return ranked[i].Symbol<ranked[j].Symbol}
  return ranked[i].Score>ranked[j].Score
 })
 if len(ranked)>topN {ranked=ranked[:topN]}
 if len(ranked)==0 {return ranked,nil}
 for i:=range ranked {ranked[i].Weight=1/float64(len(ranked))}
 return ranked,nil
}

// PaperTrade applies a virtual cash buy or sell using a given reference price.
// Fees are charged on notional turnover. It is deliberately not broker code.
func PaperTrade(cash,shares,price,quantity,feeBps float64,buy bool)(float64,float64,error){
 vals:=[]float64{cash,shares,price,quantity,feeBps}
 for _,v:=range vals{if math.IsNaN(v)||math.IsInf(v,0)||v<0{return cash,shares,errors.New("invalid trade amount")}}
 if price==0||quantity==0||feeBps>10000{return cash,shares,errors.New("invalid trade configuration")}
 notional:=price*quantity
 fee:=notional*feeBps/10000
 if buy {
  if cash+1e-8<notional+fee {return cash,shares,errors.New("insufficient paper cash")}
  return cash-notional-fee,shares+quantity,nil
 }
 if shares+1e-8<quantity{return cash,shares,errors.New("insufficient paper holdings")}
 return cash+notional-fee,shares-quantity,nil
}
