#!/usr/bin/env python3
"""Headless name-only entry and browser-workspace acceptance; no AI calls."""
import os
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

BASE = os.environ.get("INVESTONE_LIVE_URL", "http://localhost:18893").rstrip("/")
OUTPUT = Path(os.environ.get("INVESTONE_QA_OUTPUT", "/tmp/investone-name-qa"))
OUTPUT.mkdir(parents=True, exist_ok=True)
NAME = "张同学 Sam"

def access(page):
    page.goto(BASE + "/app/", wait_until="domcontentloaded")
    page.locator("#account-form").wait_for(timeout=30000)
    assert page.locator('input[type="password"]').count() == 0
    assert page.get_by_role("button", name="Create account", exact=True).count() == 0

def enter(page):
    page.get_by_label("Your name", exact=True).fill(NAME)
    page.get_by_role("button", name="Enter InvestOne →", exact=True).click()
    page.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(page.locator(".userbox")).to_contain_text(NAME)

with sync_playwright() as pw:
    browser = pw.chromium.launch(headless=True)
    headers = {"X-Forwarded-Proto": "http"} if BASE.startswith("http:") else {}
    ctx = browser.new_context(viewport={"width": 1365, "height": 950}, extra_http_headers=headers)
    page = ctx.new_page()
    errors = []
    page.on("pageerror", lambda e: errors.append(str(e)))
    access(page)
    page.screenshot(path=str(OUTPUT / "name-entry-desktop.png"), full_page=True)
    enter(page)
    page.get_by_role("link", name="Personalised Intelligence", exact=True).click()
    page.locator('#profile-form select[name="risk"]').select_option("High")
    page.locator('#profile-form button[type="submit"]').click()
    expect(page.locator("#toast")).to_contain_text("Settings saved")
    page.reload(wait_until="domcontentloaded")
    page.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(page.locator('#profile-form select[name="risk"]')).to_have_value("High")
    page.locator('.account-entry[data-action="logout"]').click()
    page.locator("#account-form").wait_for()
    page.reload(wait_until="domcontentloaded")
    page.locator("#account-form").wait_for()
    enter(page)
    page.get_by_role("link", name="Personalised Intelligence", exact=True).click()
    expect(page.locator('#profile-form select[name="risk"]')).to_have_value("High")
    print("PASS: Chinese/English name-only entry, no password/registration, refresh, sign-out and saved browser data", flush=True)
    second = browser.new_context(viewport={"width": 1280, "height": 850}, extra_http_headers=headers)
    other = second.new_page()
    access(other)
    enter(other)
    other.get_by_role("link", name="Personalised Intelligence", exact=True).click()
    expect(other.locator('#profile-form select[name="risk"]')).to_have_value("Moderate")
    print("PASS: identical names in different browsers do not share data", flush=True)
    mobile = browser.new_context(viewport={"width": 390, "height": 844}, is_mobile=True, extra_http_headers=headers)
    small = mobile.new_page()
    access(small)
    assert small.evaluate("document.documentElement.scrollWidth <= innerWidth")
    small.screenshot(path=str(OUTPUT / "name-entry-mobile.png"), full_page=True)
    enter(small)
    expect(small.locator(".account-entry")).to_be_visible()
    assert not errors, "Browser runtime errors occurred"
    print("PASS: mobile name entry and visible sign-out; no browser errors", flush=True)
    browser.close()
