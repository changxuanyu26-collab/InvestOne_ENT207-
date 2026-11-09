#!/usr/bin/env python3
"""REAL public data health report. Makes external network calls; NEVER fabricates a quote.
All numbers, dates and source links are returned by third-party providers.
"""
import csv
import datetime as dt
import io
import json
import os
import sys
import time
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET

now = dt.datetime.now(dt.timezone.utc)
UA = "InvestOne-data-check/2.1 (https://github.com/changxuanyu26-collab/InvestOne_ENT207-)"
def fetch(url, accept="*/*"):
    request = urllib.request.Request(url,headers={"User-Agent":UA,"Accept":accept})
    with urllib.request.urlopen(request,timeout=18) as response:
        if response.status!=200:raise RuntimeError(f"HTTP {response.status}")
        return response.read(8_000_000)

def valid(x):
    try:return 0<float(x)<1e10
    except (ValueError,TypeError,OverflowError):return False

def yahoo():
    errors=[]
    for host in ("query1.finance.yahoo.com","query2.finance.yahoo.com"):
        url=f"https://{host}/v8/finance/chart/SPY?range=1mo&interval=1d"
        try:
            obj=json.loads(fetch(url,"application/json"))["chart"]["result"][0]
            meta=obj["meta"];price=meta.get("regularMarketPrice")
            bars=list(zip(obj.get("timestamp",[]),obj["indicators"]["quote"][0]["close"]))
            good=[int(t) for t,c in bars if valid(c)]
            if not valid(price) or not good:raise ValueError("no usable quote or price bars")
            t=int(meta.get("regularMarketTime") or max(good))
            age=(now-dt.datetime.fromtimestamp(t,dt.timezone.utc)).total_seconds()/86400
            if age>14 or age < -1:raise ValueError(f"stale/future quote: age {age:.1f} days")
            return {"ok":True,"source":host,"symbol":"SPY","currency":meta.get("currency"),
                    "price":price,"as_of":dt.datetime.fromtimestamp(t,dt.timezone.utc).isoformat(),
                    "historical_bars":len(good),"url":url,"category":"unofficial public market data"}
        except Exception as e:errors.append(f"{host}: {type(e).__name__}: {e}")
    raise RuntimeError(" | ".join(errors))

def stooq():
    end=now.date();start=end-dt.timedelta(days=32)
    url="https://stooq.com/q/d/l/?"+urllib.parse.urlencode({
        "s":"spy.us","i":"d","d1":start.strftime("%Y%m%d"),"d2":end.strftime("%Y%m%d")})
    text=fetch(url,"text/csv").decode("utf-8-sig",errors="replace")
    rows=list(csv.DictReader(io.StringIO(text)))
    bars=[(dt.date.fromisoformat(r["Date"]),float(r["Close"])) for r in rows if r.get("Date") and valid(r.get("Close"))]
    if len(bars)<2:raise ValueError("Stooq refused or returned no daily data")
    last=max(b[0] for b in bars)
    if (end-last).days>14 or last>end+dt.timedelta(days=1):raise ValueError("date stale/future")
    return {"ok":True,"source":"Stooq","symbol":"SPY","price":bars[-1][1],
            "currency":"USD","as_of":last.isoformat(),"historical_bars":len(bars),"url":url,
            "category":"historical end-of-day close, not a real-time quote"}

def tencent():
    url="https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?"+urllib.parse.urlencode(
        {"param":"sh600519,day,,,35,qfq"})
    body=fetch(url,"application/json").decode("utf-8-sig",errors="replace").strip()
    if not body.startswith("{"):body=body.partition("=")[2].strip().rstrip(";")
    node=json.loads(body)["data"]["sh600519"]
    rows=node.get("qfqday") or node.get("day") or []
    bars=[(dt.date.fromisoformat(r[0]),float(r[2])) for r in rows if len(r)>2 and valid(r[2])]
    if len(bars)<2:raise ValueError("Tencent did not return daily prices")
    last=max(d for d,c in bars)
    if (now.date()-last).days>14 or last>now.date()+dt.timedelta(days=1):
        raise ValueError(f"stale/future A-share bars: {last}")
    return {"ok":True,"source":"Tencent Finance","symbol":"600519.SS",
            "price":bars[-1][1],"currency":"CNY","as_of":last.isoformat(),
            "historical_bars":len(bars),"url":url,
            "category":"unofficial A-share daily history"}

def news():
    url="https://news.google.com/rss/search?"+urllib.parse.urlencode({
        "q":"NVDA earnings stock when:14d","hl":"en-US","gl":"US","ceid":"US:en"})
    items=ET.fromstring(fetch(url,"application/rss+xml")).findall("./channel/item")
    if not items:raise ValueError("empty news RSS")
    first=items[0];title=(first.findtext("title") or "").strip()
    link=(first.findtext("link") or "").strip()
    if not title or not link.startswith("https://"):raise ValueError("RSS title/link missing")
    return {"ok":True,"source":"Google News RSS","symbol":"NVDA",
            "article_count":len(items),"first_title":title[:180],"first_link":link,
            "url":url,"category":"headline links, not verified full articles"}

def sec():
    email=os.environ.get("INVESTONE_SEC_CONTACT_EMAIL","").strip()
    if "@" not in email:return {"ok":False,"skipped":True,"reason":"Set identifying SEC contact email before live EDGAR access"}
    url="https://www.sec.gov/files/company_tickers.json"
    req=urllib.request.Request(url,headers={"User-Agent":f"InvestOne research contact {email}"})
    with urllib.request.urlopen(req,timeout=15) as response:companies=json.load(response)
    matches=[v for v in companies.values() if v.get("ticker")=="NVDA"]
    if not matches:raise ValueError("No SEC CIK for NVDA")
    return {"ok":True,"source":"SEC EDGAR","symbol":"NVDA",
            "cik":matches[0]["cik_str"],"url":url,"category":"official SEC company mapping"}

checks={"yahoo_spy":yahoo,"stooq_spy":stooq,"tencent_600519":tencent,
        "google_news_nvda":news,"sec_nvda":sec}
report={"checked_at_utc":now.isoformat(),
        "mode":"REAL_NETWORK_ONLY_NO_MOCK_OR_RECORDED_PRICE_FALLBACK","checks":{}}
for key,fn in checks.items():
    start=time.monotonic()
    try:entry=fn()
    except Exception as e:entry={"ok":False,"error":f"{type(e).__name__}: {str(e)[:400]}"}
    entry["duration_s"]=round(time.monotonic()-start,2)
    report["checks"][key]=entry
    print(key, "PASS" if entry.get("ok") else "SKIP" if entry.get("skipped") else "FAIL",
          entry.get("source") or entry.get("error") or entry.get("reason"),flush=True)
with open("live-data-report.json","w",encoding="utf-8") as file:
    json.dump(report,file,indent=2,ensure_ascii=False)
c=report["checks"]
sys.exit(0 if ((c["yahoo_spy"]["ok"] or c["stooq_spy"]["ok"]) and
              c["tencent_600519"]["ok"] and c["google_news_nvda"]["ok"]) else 1)
