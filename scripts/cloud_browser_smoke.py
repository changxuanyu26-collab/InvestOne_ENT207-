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
 page.locator("nav .nav-item").first.wait_for(timeout=40000)
 actual=page.locator("nav .nav-item").all_text_contents()
 expected=["AI Investment Chatbot","Personalised Intelligence","Continuous Automation","Community Integration","Investor Digital Humans"]
 assert all(any(e in item for item in actual) for e in expected),actual
 assert page.get_by_role("link",name="＋ Connect Highway AI Key").count()==1
 page.get_by_role("link",name="Investor Digital Humans").click()
 page.locator(".persona-card").first.wait_for(timeout=10000)
 assert page.locator(".persona-card").count()==3
 assert page.locator(".avatar-mouth").count()==3
 page.get_by_role("button",name="Analyze with this framework →").first.click()
 assert "chat" in page.url
 page.locator("textarea#chat-input").wait_for(timeout=10000)
 assert not errors,errors
 print("DESKTOP PASS: landing, CTA, guest entry, 5 modules, key-setup link, digital avatars",flush=True)
 ctx.close()
 mobile=browser.new_context(viewport={"width":390,"height":844},is_mobile=True,device_scale_factor=2)
 p=mobile.new_page()
 p.goto(BASE,wait_until="load",timeout=60000)
 assert p.get_by_role("link",name="Try InvestOne live web app").is_visible()
 assert p.locator("h1").count()==1
 p.get_by_role("link",name="Try InvestOne live web app").click()
 p.locator("nav .nav-item").first.wait_for(timeout=40000)
 print("MOBILE PASS: one-button, no-account live app entry",flush=True)
 mobile.close()
 browser.close()
