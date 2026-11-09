#!/usr/bin/env python3
"""Verify the actual Railway deployment: separate visitor data, live quotes, news, and AI-key gating."""
import http.cookiejar
import json
import os
import time
import urllib.error
import urllib.request

BASE = os.environ.get("INVESTONE_LIVE_URL","https://investone-live-production.up.railway.app").rstrip("/")

class Browser:
    def __init__(self):
        self.jar=http.cookiejar.CookieJar()
        self.opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
    def get(self,path):
        req=urllib.request.Request(BASE+path,headers={"X-InvestOne":"1","User-Agent":"InvestOne-Cloud-QA/1.0"})
        with self.opener.open(req,timeout=70) as r: return json.loads(r.read())
    def post(self,path,body):
        req=urllib.request.Request(BASE+path,json.dumps(body).encode(),headers={"X-InvestOne":"1","Content-Type":"application/json","Origin":BASE,"User-Agent":"InvestOne-Cloud-QA/1.0"})
        with self.opener.open(req,timeout=70) as r:return json.loads(r.read())

for attempt in range(16):
    try:
        test=Browser()
        health=test.get("/api/health")
        assert health.get("app")=="InvestOne" and health.get("mode")=="CLOUD_LIVE_BACKEND",health
        break
    except Exception as ex:
        if attempt==15:raise
        print("Cloud readiness retry:",attempt+1,type(ex).__name__,flush=True)
        time.sleep(10)
print("HEALTH PASS",health,flush=True)
a,b=Browser(),Browser()
x,y=a.get("/api/state"),b.get("/api/state")
assert x["state"]["profile"]["risk"]==y["state"]["profile"]["risk"]
a.post("/api/profile",{"risk":"Conservative","horizon":"3–5 years","knowledge":"Beginner","interests":"Technology","email":"","model":"deepseek/deepseek-v4.1-flash","max_tokens":8192})
x,y=a.get("/api/state"),b.get("/api/state")
assert x["state"]["profile"]["risk"]=="Conservative"
assert y["state"]["profile"]["risk"]!="Conservative"
print("VISITOR-ISOLATION PASS",flush=True)

qresult=a.get("/api/quotes?symbols=SPY&range=1mo")
assert qresult.get("quotes"),qresult
q=qresult["quotes"][0]
assert q["symbol"]=="SPY" and float(q["price"])>0 and q.get("source") and q.get("as_of") and q.get("retrieved") and q.get("points"),q
assert not qresult.get("errors",{}).get("SPY"),qresult
print("REAL MARKET PASS:",{k:q[k] for k in ("symbol","price","currency","source","as_of","retrieved")},flush=True)

research=a.get("/api/research?symbol=NVDA")
news=research.get("news",[])
assert news,research.get("errors")
assert all(item.get("title") and item.get("url") for item in news[:3])
print("REAL NEWS PASS:",len(news),flush=True)
try:
    response=a.post("/api/chat",{"message":"Summarize NVDA","symbol":"NVDA","persona":"assistant","include_community":False})
    raise AssertionError("Unexpected paid AI success without a key: "+str(response)[:100])
except urllib.error.HTTPError as ex:
    assert ex.code in (400,401,422,502) and b"Highway API key" in ex.read(),ex.code
print("AI-KEY GUARD PASS (no paid AI request made)",flush=True)

# Exercise persistent automation with a non-triggering rule, without using AI credits.
a.post("/api/rules",{"type":"price_below","symbol":"SPY","threshold":1,"keyword":"","auto_ai":False})
after=a.get("/api/state")
assert any(r.get("symbol")=="SPY" and r.get("threshold")==1 for r in after["state"]["rules"]),after["state"]["rules"]
check=a.post("/api/rules/check",{})
assert check.get("ok") is True,check
print("CLOUD MONITORING PASS: rule persisted, real provider check executed",flush=True)

# Community responses come from the public upstream endpoint, or report its error.
community=a.get("/api/community?symbol=NVDA")
assert isinstance(community.get("items"),list) and isinstance(community.get("error"),str),community
print("COMMUNITY:",len(community["items"]),"actual items; error:",community["error"][:160] or "none",flush=True)

# Use actual historical bars for the full ETF universe: no synthetic backtest curve.
quant=a.post("/api/quant",{"universe":"sectors","range":"2y","top_n":3,"fee_bps":5,"momentum":50,"low_vol":30,"trend":20,"risk_off":True})
assert quant.get("curve") and quant.get("sources") and quant.get("observations",0)>100,quant
print("REAL MULTI-FACTOR BACKTEST PASS:",quant["observations"],"observations,",len(quant["sources"]),"sources",flush=True)
print("VERIFIED: REAL BACKEND, ISOLATED VISITORS, MARKET DATA AND NEWS",flush=True)
