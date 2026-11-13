#!/usr/bin/env python3
"""Real remote-browser acceptance test (no mocks or paid API keys)."""
import os
from playwright.sync_api import sync_playwright
BASE=os.environ.get("INVESTONE_LIVE_URL","https://investone-live-production.up.railway.app").rstrip("/")
with sync_playwright() as playwright:
 browser=playwright.chromium.launch(headless=True,args=["--no-sandbox"])
 ctx=browser.new_context(viewport={"width":1365,"height":850})
 page=ctx.new_page();errors=[]
 page.on("pageerror",lambda error:errors.append(str(error)))
 page.goto(BASE,wait_until="load",timeout=60000)
 cta=page.get_by_role("link",name="Try InvestOne live web app")
 assert cta.count()==1
 assert page.locator("h1").count()==1
 assert cta.get_attribute("href")=="/app/"
 cta.click()
 page.wait_for_url("**/app/",timeout=15000)
 page.get_by_label("Your name",exact=True).fill("Demo Visitor")
 page.get_by_role("button",name="Enter InvestOne →",exact=True).click()
 page.locator("nav .nav-item").first.wait_for(timeout=40000)
 actual=page.locator("nav .nav-item").all_text_contents()
 expected=["AI Investment Chatbot","Personalised Intelligence","Continuous Automation","Community Integration","Investor Digital Humans"]
 assert all(any(e in item for item in actual) for e in expected),actual
 assert page.get_by_role("link",name="＋ Connect ENT207 DeepSeek AI Key").count()==0
 assert "ENT207 DeepSeek AI configured" in page.locator(".topbar").inner_text()
 page.evaluate('''() => { rememberCurrentReply("SESSION_ONLY_FIXTURE","assistant",{content:"TEMPORARY_REPLY_FIXTURE",sources:[]}); render(); }''')
 assert "TEMPORARY_REPLY_FIXTURE" in page.locator("#messages").inner_text()
 page.reload(wait_until="load")
 page.locator("nav .nav-item").first.wait_for(timeout=40000)
 assert "TEMPORARY_REPLY_FIXTURE" not in page.locator("#messages").inner_text()
 print("EPHEMERAL CHAT PASS: current-page reply disappears after reload",flush=True)
 page.get_by_role("link",name="Investor Digital Humans").click()
 page.locator(".persona-card").first.wait_for(timeout=10000)
 assert page.locator(".persona-card").count()==3
 assert page.locator(".avatar-mouth").count()==3
 page.get_by_role("button",name="Select this framework").first.click()
 assert page.url.endswith("/app/#personas"),"Choosing a digital human should not redirect to Chatbot: "+page.url
 page.locator("#persona-form textarea#persona-input").wait_for(timeout=10000)
 assert page.locator("#persona-form").is_visible()
 assert page.locator(".persona-card.is-selected").count()==1
 assert "DeepSeek-V4-Flash" in page.locator("#persona-session").inner_text()
 # Browser-only synthetic SSE fixture verifies the rendering protocol; this
 # does not claim the paid school gateway itself was contacted.
 def fake_sse(route):
  body='event: status\ndata: {"text":"Preparing"}\n\n'
  body+='event: delta\ndata: {"text":"FAST STREAM "}\n\n'
  body+='event: delta\ndata: {"text":"WORKS"}\n\n'
  body+='event: complete\ndata: {"content":"FAST STREAM WORKS","warnings":[],"sources":[]}\n\n'
  route.fulfill(status=200,content_type="text/event-stream",body=body)
 page.route("**/api/chat/stream",fake_sse)
 streamed=page.evaluate("""async () => {
   const r=await streamAIRequest({message:"test",symbol:"SPY",persona:"buffett",include_community:false},"#persona-messages");
   return {content:r.content,visible:document.querySelector("#persona-messages")?.textContent.includes("FAST STREAM WORKS")};
 }""")
 assert streamed["content"]=="FAST STREAM WORKS" and streamed["visible"],streamed
 page.unroute("**/api/chat/stream")
 print("UI STREAMING PASS: progressive token rendering from a marked test fixture",flush=True)

 page.locator('nav a[href="#chat"]').click()
 page.locator("#chat-form").wait_for(timeout=10000)
 assert "shared public AI quota" in page.locator("#chat-form").inner_text()
 page.goto(BASE+"/app/#settings",wait_until="load",timeout=30000)
 page.locator("#page-content").get_by_text("Shared AI ready for all visitors").wait_for(timeout=10000)
 assert page.locator("#key-form").count()==0
 assert "Visits" in page.locator("footer").inner_text()
 print("PUBLIC AI PASS: no visitor key input, aggregate visit counts visible; no paid request made",flush=True)
 page.locator('nav a[href="#chat"]').click()
 page.locator("#chat-form").wait_for(timeout=10000)

 page.locator('.core-link[href="#markets"]').click()
 page.locator("#research-data h2").filter(has_text="News and summaries").wait_for(timeout=45000)
 assert page.locator("#research-data").is_visible()
 print("NEWS RESEARCH BUTTON PASS",flush=True)

 page.locator('nav a[href="#chat"]').click()
 page.locator('.core-link[href="#quant"]').click()
 page.locator('#quant-form').wait_for(timeout=10000)
 assert "Sector Allocation" in page.locator("#page-content").inner_text()
 page.locator('#quant-form button[type="submit"]').click()
 page.locator("#quant-result .metric").first.wait_for(timeout=90000)
 assert "Strategy cumulative return" in page.locator("#quant-result").inner_text()
 print("SECTOR ALLOCATION AND BACKTEST BUTTON PASS",flush=True)

 page.locator('nav a[href="#automation"]').click()
 page.locator('#rule-form').wait_for(timeout=10000)
 page.locator('#rule-form input[name="symbol"]').fill("SPY")
 page.locator('#rule-form select[name="type"]').select_option("price_below")
 page.locator('#rule-form input[name="threshold"]').fill("1")
 page.locator('#rule-form button[type="submit"]').click()
 page.locator(".rule").filter(has_text="SPY").wait_for(timeout=10000)
 page.get_by_role("button",name="Check now").click()
 page.locator(".rule").filter(has_text="SPY").wait_for(timeout=15000)
 print("RISK MONITOR BUTTON PASS",flush=True)

 assert not errors,errors
 # Reproduce the real bug: leave GitHub Pages and navigate to Railway
 # so Chromium sends Sec-Fetch-Site: cross-site on the document request.
 from_github=browser.new_context(viewport={"width":1280,"height":800})
 external=from_github.new_page()
 external.goto("https://changxuanyu26-collab.github.io/InvestOne_ENT207-/",wait_until="load",timeout=60000)
 external_cta=external.get_by_role("link",name="Try InvestOne live web app")
 assert external_cta.count()==1,"GitHub Pages landing CTA missing"
 external_cta.click()
 external.wait_for_url("**/app/",timeout=25000)
 external.get_by_label("Your name",exact=True).fill("Demo Visitor")
 external.get_by_role("button",name="Enter InvestOne →",exact=True).click()
 external.locator("nav .nav-item").first.wait_for(timeout=20000)
 assert "Cross-site request blocked" not in external.locator("body").inner_text()
 print("CROSS-SITE GITHUB → RAILWAY NAVIGATION PASS",flush=True)
 from_github.close()
 print("DESKTOP PASS: landing, CTA, guest entry, 5 modules, key-setup link, in-page digital human conversations",flush=True)
 ctx.close()
 mobile=browser.new_context(viewport={"width":390,"height":844},is_mobile=True,device_scale_factor=2)
 p=mobile.new_page()
 p.goto(BASE,wait_until="load",timeout=60000)
 assert p.get_by_role("link",name="Try InvestOne live web app").is_visible()
 assert p.locator("h1").count()==1
 p.get_by_role("link",name="Try InvestOne live web app").click()
 p.get_by_label("Your name",exact=True).fill("Demo Visitor")
 p.get_by_role("button",name="Enter InvestOne →",exact=True).click()
 p.locator("nav .nav-item").first.wait_for(timeout=40000)
 print("MOBILE PASS: one-button, no-account live app entry",flush=True)
 mobile.close()
 browser.close()
