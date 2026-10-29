#!/usr/bin/env python3
"""Render a genuine 63-second narrated MP4 using the offline InvestOne site.

Captures actual local UI interactions in Chromium. Prices and AI responses
are clearly labeled synthetic. This is not a recording of real live data.
Audio and video are encoded separately to avoid platform encoder issues.
"""
import asyncio
import json
import pathlib
import subprocess
import tempfile
from playwright.async_api import async_playwright

ROOT = pathlib.Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
SCENES = [
 ("overview","A clearer investment workspace","Welcome to InvestOne. One place to research markets, understand portfolios, and explore investment ideas."),
 ("markets","Market research, made practical","Explore example prices, candlestick charts, company context, and an evidence-first research workspace."),
 ("portfolio","Practice without real money","Paper trading helps you test investment decisions with virtual cash and transparent simulated costs."),
 ("quant","Research systematic strategies","Quant Lab explores factor configurations and portfolio methods. All curves in this demo are synthetic illustrations."),
 ("ai","Ask better research questions","The AI research workspace organizes investment questions, source quality, valuation risk, and portfolio context."),
 ("automation","Define monitoring rules","Create an example price alert and understand how monitoring works when the local desktop service is active."),
 ("community","Balance opposing viewpoints","Compare bullish and bearish arguments without confusing investor sentiment with verified financial disclosures."),
 ("frameworks","Learn from established principles","Explore educational frameworks inspired by Buffett, Munger, and Graham, with clear distinctions from real people."),
 ("overview","One connected research journey","InvestOne brings research, simulated portfolios, quantitative tools, and monitoring together. This walkthrough uses fictional data.")
]
DURATION = 7

def run(args):
    cp = subprocess.run(args, capture_output=True, text=True)
    if cp.returncode:
        print("COMMAND FAILED:", " ".join(map(str,args)), flush=True)
        print("STDOUT:", cp.stdout[-4000:], flush=True)
        print("STDERR:", cp.stderr[-8000:], flush=True)
        raise RuntimeError(f"Process exited {cp.returncode}")
    return cp

async def record(folder):
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True, args=["--no-sandbox"])
        context = await browser.new_context(viewport={"width":1360,"height":765}, device_scale_factor=1, reduced_motion="reduce")
        page = await context.new_page()
        html = (DOCS/"index.html").read_text(encoding="utf-8")
        css = (DOCS/"site.css").read_text(encoding="utf-8")
        js = (DOCS/"site.js").read_text(encoding="utf-8")
        html = html.replace('<link rel="stylesheet" href="site.css">', "<style>"+css+"</style>")
        html = html.replace('<script src="site.js"></script>', "<script>"+js+"</script>")
        errors = []
        page.on("pageerror", lambda err: errors.append(str(err)))
        await page.set_content(html, wait_until="load")
        await page.locator('[data-go="overview"]').first.wait_for(timeout=15000)
        await page.evaluate("""() => {
          let style=document.createElement('style');
          style.innerText='.video-caption{position:fixed;left:262px;right:22px;bottom:12px;z-index:9999;background:rgba(20,31,47,.95);color:#fff;padding:14px 19px;border-radius:10px;box-shadow:0 8px 25px #0003;font:14px Arial,sans-serif;pointer-events:none}.video-caption b{font-size:19px;display:block;margin-bottom:5px}.video-caption span{font-size:14px;color:#e1e7f0}.video-caption small{float:right;color:#f5c781}';
          document.head.appendChild(style);
          let caption=document.createElement('div');caption.id='video-caption';caption.className='video-caption';document.body.appendChild(caption);
        }""")
        for i,(tab,title,words) in enumerate(SCENES):
            if i:
                await page.locator(f'[data-go="{tab}"]').first.click(timeout=10000)
            if tab == "portfolio":
                await page.locator('#trade-form select[name="symbol"]').select_option("MSFT")
                await page.locator('#trade-form input[name="quantity"]').fill("3")
                await page.locator('#trade-form button[type="submit"]').click()
            if tab == "quant":
                await page.locator("#factor-select").select_option("lowvol")
                await page.locator("#quant-run").click()
            if tab == "ai":
                await page.locator("#ask-form button").click()
            if tab == "automation":
                await page.locator('#rule-form input[name="threshold"]').fill("120")
                await page.locator("#rule-form button").click()
            await page.evaluate("""({title,words,index})=>{
              const b=document.getElementById('video-caption');b.replaceChildren();
              const h=document.createElement('b');h.textContent=title;
              const sub=document.createElement('span');sub.textContent=words;
              const note=document.createElement('small');note.textContent=(index+1)+'/9  ·  SYNTHETIC DATA';
              b.append(note,h,sub);
            }""", {"title":title,"words":words,"index":i})
            await page.screenshot(path=str(folder / f"frame_{i:02}.png"), full_page=False, animations="disabled")
            print(f"Screenshot {i+1}/9: {tab}",flush=True)
        await browser.close()
        if errors:raise RuntimeError("Browser JavaScript errors: "+str(errors))

def create_audio(folder):
    clips=[]
    for i,(_,_,words) in enumerate(SCENES):
        raw=folder/f"speech_{i:02}.wav"; padded=folder/f"speech_{i:02}_pad.wav"
        run(["espeak","-v","en-us+f3","-s","158","-p","49","-a","160","-w",str(raw),words])
        run(["ffmpeg","-y","-hide_banner","-loglevel","error","-i",str(raw),
             "-af",f"apad=whole_dur={DURATION},atrim=0:{DURATION},aresample=44100",
             "-ar","44100","-ac","1",str(padded)])
        clips.append(padded)
    playlist=folder/"audio.ffconcat"
    playlist.write_text("ffconcat version 1.0\n"+"".join(f"file '{x.as_posix()}'\n" for x in clips),encoding="utf-8")
    output=folder/"narration.wav"
    run(["ffmpeg","-y","-hide_banner","-loglevel","error","-safe","0","-f","concat","-i",str(playlist),"-c:a","pcm_s16le",str(output)])
    return output

def create_video(folder):
    clips=[]
    for i in range(len(SCENES)):
        image=folder/f"frame_{i:02}.png";clip=folder/f"clip_{i:02}.mp4"
        run(["ffmpeg","-y","-hide_banner","-loglevel","error",
             "-threads","1","-filter_threads","1",
             "-loop","1","-framerate","24","-t",str(DURATION),"-i",str(image),
             "-vf","pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p","-c:v","libx264","-preset","ultrafast",
             "-crf","28","-pix_fmt","yuv420p","-r","24","-an",str(clip)])
        print(f"Encoded segment {i+1}/9",flush=True)
        clips.append(clip)
    playlist=folder/"video.ffconcat"
    playlist.write_text("ffconcat version 1.0\n"+"".join(f"file '{x.as_posix()}'\n" for x in clips),encoding="utf-8")
    merged=folder/"silent.mp4"
    run(["ffmpeg","-y","-hide_banner","-loglevel","error",
         "-safe","0","-f","concat","-i",str(playlist),"-c","copy",str(merged)])
    return merged

def verify(path):
    info=json.loads(run(["ffprobe","-v","error","-show_entries",
        "format=duration:stream=codec_type,codec_name","-of","json",str(path)]).stdout)
    types={s["codec_type"] for s in info["streams"]}
    duration=float(info["format"]["duration"])
    if types!={"audio","video"} or not 55<duration<75 or path.stat().st_size<100000:
        raise RuntimeError("Invalid output video: "+str(info))
    print(f"VERIFIED video={path} duration={duration:.2f}s tracks={types} size={path.stat().st_size}",flush=True)

async def main():
    with tempfile.TemporaryDirectory(prefix="investone-film-") as tmp:
        folder=pathlib.Path(tmp)
        await record(folder)
        audio=create_audio(folder)
        video=create_video(folder)
        out=DOCS/"demo.mp4"
        run(["ffmpeg","-y","-hide_banner","-loglevel","error",
             "-i",str(video),"-i",str(audio),"-c:v","copy",
             "-c:a","aac","-b:a","112k","-ar","44100",
             "-movflags","+faststart","-shortest",str(out)])
        verify(out)

if __name__=="__main__":
    asyncio.run(main())
