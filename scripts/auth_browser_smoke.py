#!/usr/bin/env python3
"""Headless account acceptance against a running cloud server; never prints credentials."""
import os
import secrets
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

BASE = os.environ.get("INVESTONE_LIVE_URL", "http://localhost:18892").rstrip("/")
OUTPUT = Path(os.environ.get("INVESTONE_QA_OUTPUT", "/tmp/investone-auth-qa"))
OUTPUT.mkdir(parents=True, exist_ok=True)
PASSWORD = secrets.token_urlsafe(24)
USERNAME = "qa_" + secrets.token_hex(6)

def access(page):
    page.goto(BASE + "/app/", wait_until="domcontentloaded")
    page.locator("#account-form").wait_for(timeout=30000)

def credentials(page, password=PASSWORD):
    page.locator('#account-form input[name="username"]').fill(USERNAME)
    page.locator('#account-form input[name="password"]').fill(password)

with sync_playwright() as pw:
    browser = pw.chromium.launch(headless=True)
    headers = {"X-Forwarded-Proto": "http"} if BASE.startswith("http:") else {}
    ctx = browser.new_context(viewport={"width": 1365, "height": 950}, extra_http_headers=headers)
    page = ctx.new_page()
    errors = []
    page.on("pageerror", lambda e: errors.append(str(e)))
    access(page)
    page.screenshot(path=str(OUTPUT / "login-desktop.png"), full_page=True)
    page.get_by_role("button", name="Continue as guest →").click()
    page.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(page.locator(".userbox")).to_contain_text("Guest workspace")
    page.get_by_role("link", name="Personalised Intelligence", exact=True).click()
    page.locator('#profile-form select[name="risk"]').select_option("High")
    page.locator('#profile-form button[type="submit"]').click()
    expect(page.locator("#toast")).to_contain_text("Settings saved")
    page.locator('.account-entry[data-action="open-sign-in"]').click()
    page.get_by_role("button", name="Create account", exact=True).click()
    credentials(page)
    page.locator('input[name="confirm"]').fill("mismatched-password-value")
    page.locator('#account-form button[type="submit"]').click()
    expect(page.locator("#auth-error")).to_contain_text("Passwords do not match")
    credentials(page)
    page.locator('input[name="confirm"]').fill(PASSWORD)
    page.locator('#account-form button[type="submit"]').click()
    page.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(page.locator(".userbox")).to_contain_text(USERNAME)
    page.get_by_role("link", name="Personalised Intelligence", exact=True).click()
    expect(page.locator('#profile-form select[name="risk"]')).to_have_value("High")
    page.reload(wait_until="domcontentloaded")
    page.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(page.locator(".userbox")).to_contain_text(USERNAME)
    page.locator('.account-entry[data-action="logout"]').click()
    page.locator("#account-form").wait_for()
    page.reload(wait_until="domcontentloaded")
    page.locator("#account-form").wait_for()
    credentials(page, "incorrect-password-value")
    page.locator('#account-form button[type="submit"]').click()
    expect(page.locator("#auth-error")).to_contain_text("Incorrect username or password")
    assert page.locator("#account-form").is_visible()
    credentials(page)
    page.locator('#account-form button[type="submit"]').click()
    page.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(page.locator(".userbox")).to_contain_text(USERNAME)
    print("PASS: guest entry, registration, guest data copy, refresh, sign-out, wrong password and return login", flush=True)

    second = browser.new_context(viewport={"width": 1280, "height": 850}, extra_http_headers=headers)
    other = second.new_page()
    access(other)
    credentials(other)
    other.locator('#account-form button[type="submit"]').click()
    other.locator("nav .nav-item").first.wait_for(timeout=30000)
    other.get_by_role("link", name="Personalised Intelligence", exact=True).click()
    expect(other.locator('#profile-form select[name="risk"]')).to_have_value("High")
    # Sign-out on one device must not invalidate the other device.
    page.locator('.account-entry[data-action="logout"]').click()
    other.reload(wait_until="domcontentloaded")
    other.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(other.locator(".userbox")).to_contain_text(USERNAME)
    print("PASS: cross-device saved workspace and independent sign-out", flush=True)

    mobile = browser.new_context(viewport={"width": 390, "height": 844}, is_mobile=True, extra_http_headers=headers)
    small = mobile.new_page()
    access(small)
    assert small.evaluate("document.documentElement.scrollWidth <= innerWidth")
    small.screenshot(path=str(OUTPUT / "login-mobile.png"), full_page=True)
    small.get_by_role("button", name="Create account", exact=True).click()
    expect(small.locator('input[name="confirm"]')).to_be_visible()
    assert small.evaluate("document.documentElement.scrollWidth <= innerWidth")
    small.get_by_role("button", name="Continue as guest →").click()
    small.locator("nav .nav-item").first.wait_for(timeout=30000)
    expect(small.locator(".account-entry")).to_be_visible()
    assert not errors, "Browser runtime error(s) occurred"
    print("PASS: mobile login, registration, guest access and visible account control; no browser errors", flush=True)
    browser.close()
