#!/usr/bin/env python3
"""Browser smoke tests for the five-core static InvestOne demo.

The public demo is intentionally simulated. Tests verify working controls,
cross-module navigation and that the original product video is preserved.
"""
import functools
import http.server
import pathlib
import threading
import urllib.request

from playwright.sync_api import sync_playwright

ROOT=pathlib.Path(__file__).resolve().parents[1]
DOCS=ROOT/"docs"

class Quiet(http.server.SimpleHTTPRequestHandler):
    def log_message(self,*args): pass

def run():
    handler=functools.partial(Quiet,directory=str(DOCS))
    server=http.server.ThreadingHTTPServer(("127.0.0.1",0),handler)
    threading.Thread(target=server.serve_forever,daemon=True).start()
    address=f"http://127.0.0.1:{server.server_port}/"
    errors=[]
    try:
        with sync_playwright() as p:
            browser=p.chromium.launch(headless=True,args=["--no-sandbox"])
            page=browser.new_page(viewport={"width":1440,"height":900})
            page.on("pageerror",lambda error:errors.append(str(error)))
            page.goto(address,wait_until="load")
            page.locator('#main-nav button').first.wait_for(timeout=12000)
            expected=["AI Investment Chatbot","Personalised Intelligence","Continuous Automation","Community Integration","Investor Digital Humans"]
            actual=page.locator("#main-nav .nav-text").all_text_contents()
            assert actual==expected, f"Navigation mismatch: {actual!r}"
            assert page.locator('#page-content h1').count()==1
            assert "Research starts" in page.locator('#page-content h1').inner_text()

            page.locator('#main-nav [data-go="personal"]').click()
            page.locator('#profile-form select[name="risk"]').select_option("Conservative")
            page.locator('#profile-form select[name="horizon"]').select_option("Long term")
            page.locator('#profile-form button[type="submit"]').click()
            assert "profile updated" in page.locator("#profile-feedback").inner_text().lower()

            page.locator('#main-nav [data-go="ai"]').click()
            assert "Conservative" in page.locator('.context-strip').inner_text()
            page.locator('#ask-form input[name="question"]').fill("Assess my NVIDIA exposure")
            page.locator('#ask-form button[type="submit"]').click()
            assert "Conservative".lower() in page.locator('#messages').inner_text().lower()
            assert "SIMULATED AI RESPONSE" in page.locator("#messages").inner_text()

            page.locator('#main-nav [data-go="automation"]').click()
            page.locator('#rule-form select[name="symbol"]').select_option("NVDA")
            page.locator('#rule-form input[name="threshold"]').fill("120")
            page.locator('#rule-form button[type="submit"]').click()
            assert "NVDA below $120.00" in page.locator("#page-content").inner_text()

            page.locator('#main-nav [data-go="community"]').click()
            page.locator('[data-prompt*="bullish"]').click()
            assert "bullish" in page.locator("#messages").inner_text().lower()
            assert "AI Investment Chatbot" in page.locator("#crumb").inner_text()

            page.locator('#main-nav [data-go="humans"]').click()
            page.locator('[data-framework="Graham"]').click()
            assert "Graham-inspired review" in page.locator("#page-content").inner_text()
            page.locator('[data-prompt*="Graham"]').click()
            assert "Graham" in page.locator("#messages").inner_text()

            video=page.locator("#product-video")
            assert video.count()==1
            assert video.locator("source").get_attribute("src")=="demo.mp4"
            page.locator("#open-video-top").click()
            assert "open" in page.locator("#video-dialog").get_attribute("class")
            page.locator("#close-video").click()
            assert "open" not in page.locator("#video-dialog").get_attribute("class")

            with urllib.request.urlopen(address+"demo.mp4",timeout=20) as response:
                head=response.read(32)
                assert response.status==200 and b"ftyp" in head, "Original MP4 not present"
            assert not errors, f"Browser exceptions: {errors!r}"
            browser.close()
        print("PASS: five navigation items, profile -> AI, rules, community -> AI, framework -> AI, original MP4")
    finally:
        server.shutdown()

if __name__=="__main__":run()
