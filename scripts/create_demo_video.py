#!/usr/bin/env python3
"""Create an actual MP4 video from real browser interactions with the OFFLINE demo.

The browser page uses synthetic prices and scripted AI content. The output is a
recording of the public demo, NOT evidence of live AI or real market data.
"""
from __future__ import annotations

import asyncio
import pathlib
import subprocess
import tempfile
from playwright.async_api import async_playwright

ROOT = pathlib.Path(__file__).resolve().parents[1]
PAGE = ROOT / "docs" / "index.html"
OUTPUT = ROOT / "docs" / "demo.mp4"

SCENES = [
    ("overview", "INVESTONE | YOUR INVESTMENT WORKSPACE", "Overview of research, watchlists and simulated assets."),
    ("market", "MARKET RESEARCH", "Review example symbols and separate verified evidence from opinion."),
    ("portfolio", "PAPER TRADING", "Buy and sell with virtual money, never real brokerage funds."),
    ("quant", "QUANT LAB", "Explore factor signals and a synthetic strategy visualization."),
    ("chat", "AI RESEARCH ASSISTANT", "Preview the workflow using a clearly scripted, offline response."),
    ("monitor", "MONITORING & AUTOMATION", "Set price-alert rules; continuous monitoring needs the desktop app."),
    ("community", "COMMUNITY INTELLIGENCE", "Compare bullish and bearish viewpoints, clearly labeled as opinions."),
    ("framework", "INVESTOR FRAMEWORKS", "Learn through Buffett-, Munger- and Graham-inspired perspectives."),
    ("overview", "INVESTONE", "One workspace for investment research and simulation.")
]

async def capture() -> None:
    if not PAGE.exists():
        raise RuntimeError(f"Missing browser demo: {PAGE}")
    with tempfile.TemporaryDirectory(prefix="investone-video-") as workspace:
        tmp = pathlib.Path(workspace)
        async with async_playwright() as playwright:
            browser = await playwright.chromium.launch(headless=True, args=["--no-sandbox"])
            page = await browser.new_page(viewport={"width": 1360, "height": 768}, device_scale_factor=1)
            await page.goto(PAGE.as_uri(), wait_until="load")
            for i, (tab, title, description) in enumerate(SCENES):
                await page.locator(f'[data-go="{tab}"]').first.click()
                if i == 2:
                    await page.locator('#trade select[name="sym"]').select_option("MSFT")
                    await page.locator('#trade input[name="qty"]').fill("2")
                    await page.locator('#trade button[type="submit"], #trade button').first.click()
                if i == 3:
                    await page.locator("#strat").select_option("blend")
                    await page.locator("#run").click()
                if i == 4:
                    await page.locator('#ask input[name="text"]').fill("What risks should I evaluate before buying NVDA?")
                    await page.locator("#ask button").click()
                if i == 5:
                    await page.locator('#rule input[name="threshold"]').fill("120")
                    await page.locator("#rule button").click()
                await page.evaluate("""({title, description, index}) => {
                    let p = document.getElementById('recording-caption');
                    if (!p) {
                        p = document.createElement('div');
                        p.id = 'recording-caption';
                        p.style.cssText = 'position:fixed;left:333px;right:20px;bottom:22px;z-index:9999;background:rgba(6,19,37,.94);border:1px solid #427f83;border-radius:13px;padding:17px 22px;color:#f1f6ff;box-shadow:0 9px 32px #0008;font-family:Arial,sans-serif;pointer-events:none';
                        document.body.appendChild(p);
                    }
                    p.innerHTML = '';
                    const h = document.createElement('div');
                    h.textContent = title;
                    h.style.cssText = 'font-size:23px;font-weight:800;letter-spacing:.3px;color:#67e3d2;margin-bottom:5px';
                    const sub = document.createElement('div');
                    sub.textContent = description;
                    sub.style.cssText = 'font-size:16px;color:#d5e3f3';
                    const small = document.createElement('div');
                    small.textContent = (index+1) + '/9  ·  EDUCATIONAL DEMO  ·  ALL DATA SYNTHETIC';
                    small.style.cssText = 'font-size:11px;letter-spacing:1px;color:#f1be81;margin-top:8px';
                    p.append(h,sub,small);
                }""", {"title":title,"description":description,"index":i})
                await page.screenshot(path=str(tmp / f"scene_{i:02d}.png"), full_page=False, animations="disabled")
            await browser.close()

        playlist = tmp / "video.ffconcat"
        content = ["ffconcat version 1.0"]
        for i in range(len(SCENES)):
            content.append(f"file '{(tmp / f'scene_{i:02d}.png').as_posix()}'")
            content.append("duration 4.8" if i < len(SCENES)-1 else "duration 5.0")
        content.append(f"file '{(tmp / f'scene_{len(SCENES)-1:02d}.png').as_posix()}'")
        playlist.write_text("\n".join(content) + "\n", encoding="utf-8")

        cmd = [
            "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
            "-safe", "0", "-f", "concat", "-i", str(playlist),
            "-vf", "fps=24,scale=1360:768,format=yuv420p",
            "-c:v", "libx264", "-preset", "veryfast", "-crf", "29",
            "-movflags", "+faststart", "-an", str(OUTPUT)
        ]
        subprocess.run(cmd, check=True)
        if not OUTPUT.is_file() or OUTPUT.stat().st_size < 5_000:
            raise RuntimeError("MP4 output missing or unexpectedly small")
        print(f"Generated {OUTPUT} ({OUTPUT.stat().st_size} bytes)")

if __name__ == "__main__":
    asyncio.run(capture())
