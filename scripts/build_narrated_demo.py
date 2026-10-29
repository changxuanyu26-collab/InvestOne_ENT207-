#!/usr/bin/env python3
"""Reproducible InvestOne product film. All visual data synthetic."""
import asyncio, pathlib, subprocess, tempfile
from playwright.async_api import async_playwright
ROOT=pathlib.Path(__file__).resolve().parents[1]
SCENES=[
("overview","A clearer investment workspace","Welcome to InvestOne. One place to research markets, understand portfolios, and explore investment ideas."),
("markets","Market research, made practical","Explore example prices, candlestick charts, company context, and an evidence-first research workspace."),
("portfolio","Practice without real money","Paper trading helps you test investment decisions with virtual cash and transparent simulated costs."),
("quant","Research systematic strategies","Quant Lab explores factor configurations and portfolio methods. All curves in this demo are synthetic illustrations."),
("ai","Ask better research questions","The AI research workspace organizes investment questions, source quality, valuation risk, and portfolio context."),
("automation","Define monitoring rules","Create an example price alert and understand how monitoring works when the local desktop service is active."),
("community","Balance opposing viewpoints","Compare bullish and bearish arguments without confusing investor sentiment with verified financial disclosures."),
("frameworks","Learn from established principles","Explore educational frameworks inspired by Buffett, Munger, and Graham, with clear distinctions from real people."),
("overview","One connected research journey","InvestOne brings research, simulated portfolios, quantitative tools, and monitoring together. This walkthrough uses fictional data.")]
DURATION=7
def run(args):subprocess.run(args,check=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
async def main():
 with tempfile.TemporaryDirectory() as tmp:
  t=pathlib.Path(tmp)
  async with async_playwright() as p:
   browser=await p.chromium.launch(headless=True,args=["--no-sandbox"])
   page=await browser.new_page(viewport={"width":1360,"height":765},device_scale_factor=1)
   html=(ROOT/"docs/index.html").read_text(encoding="utf-8")
   css=(ROOT/"docs/site.css").read_text(encoding="utf-8")
   js=(ROOT/"docs/site.js").read_text(encoding="utf-8")
   html=html.replace('<link rel="stylesheet" href="site.css">',"<style>"+css+"</style>")
   html=html.replace('<script src="site.js"></script>',"<script>"+js+"</script>")
   await page.set_content(html,wait_until="load")
   await page.locator('[data-go="overview"]').first.wait_for(timeout=15000)
   await page.add_style_tag(content=".video-caption{position:fixed;left:260px;right:20px;bottom:12px;padding:13px 18px;border-radius:9px;z-index:200;background:#192434ee;color:#fff;font-family:Arial;min-height:77px}.video-caption b{font-size:17px}.video-caption span{font-size:13px;color:#d4e1ef}")
   await page.evaluate("()=>{let el=document.createElement('div');el.className='video-caption';el.id='video-caption';document.body.appendChild(el)}")
   for i,(tab,title,speech) in enumerate(SCENES):
    if i:
     await page.locator('[data-go="'+tab+'"]').first.click(timeout=20000)
    if tab=="portfolio":
     await page.locator('#trade-form input[name="quantity"]').fill("3")
     await page.locator('#trade-form button[type="submit"]').click()
    if tab=="quant":await page.locator("#quant-run").click()
    if tab=="ai":await page.locator("#ask-form button").click()
    if tab=="automation":await page.locator("#rule-form button").click()
    await page.evaluate("({title,speech})=>{let el=document.getElementById('video-caption');el.innerHTML='';let h=document.createElement('b');h.textContent=title;let p=document.createElement('span');p.textContent=speech;el.append(h,document.createElement('br'),p)}",{"title":title,"speech":speech})
    await page.screenshot(path=str(t/f"frame_{i:02}.png"),animations="disabled")
   await browser.close()
  clips=[]
  for i,(_,_,speech) in enumerate(SCENES):
   raw=t/f"speech_{i}.wav";pad=t/f"pad_{i}.wav";clip=t/f"clip_{i}.mp4"
   run(["espeak","-v","en-us+f3","-s","158","-p","49","-a","160","-w",str(raw),speech])
   run(["ffmpeg","-y","-v","error","-i",str(raw),"-af",f"apad=whole_dur={DURATION},atrim=0:{DURATION}","-ar","44100","-ac","1",str(pad)])
   run(["ffmpeg","-y","-v","error","-loop","1","-framerate","24","-t",str(DURATION),"-i",str(t/f"frame_{i:02}.png"),"-i",str(pad),"-c:v","libx264","-preset","ultrafast","-crf","27","-pix_fmt","yuv420p","-c:a","aac","-b:a","96k","-shortest",str(clip)])
   clips.append(clip)
  playlist=t/"playlist.ffconcat";playlist.write_text("ffconcat version 1.0\n"+"".join(f"file '{c}'\n" for c in clips))
  out=ROOT/"docs/demo.mp4"
  run(["ffmpeg","-y","-v","error","-safe","0","-f","concat","-i",str(playlist),"-c","copy","-movflags","+faststart",str(out)])
  probe=subprocess.run(["ffprobe","-v","error","-show_entries","format=duration:stream=codec_type","-of","json",str(out)],capture_output=True,text=True,check=True).stdout
  import json
  obj=json.loads(probe)
  assert {"audio","video"}=={s["codec_type"] for s in obj["streams"]},probe
  assert 55<float(obj["format"]["duration"])<75,probe
  print("Validated MP4 with audio, duration:",obj["format"]["duration"],"size:",out.stat().st_size)
if __name__=="__main__":asyncio.run(main())
