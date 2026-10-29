#!/usr/bin/env python3
"""Render a narrated product film using Kokoro neural TTS and the offline InvestOne site.

Captures actual local UI interactions in Chromium. Prices and AI responses
are clearly labeled synthetic. This is not a recording of real live data.
Audio and video are encoded separately; each scene duration follows natural speech.
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
 ("overview","Investing, made easier to understand","Meet InvestOne. A more thoughtful way to explore markets, organize your investment ideas, and make sense of what matters."),
 ("markets","See the whole picture","From clear market charts to company context, everything you need for a closer look is brought together. No more jumping between tabs."),
 ("portfolio","Put your ideas to the test","Curious about a strategy? Try a simulated trade with virtual money, and see how it changes your portfolio. Without putting real money at risk."),
 ("quant","Go deeper with Quant Lab","Explore systematic investing, compare factor strategies, and see how different assumptions shape a portfolio. The chart shown here is illustrative."),
 ("ai","Ask questions. Get perspective.","Investing can be complicated. InvestOne brings your research questions, business fundamentals, and portfolio context into one conversational workspace."),
 ("automation","Stay informed, on your terms","Create alerts for the moments that matter to you. With the desktop app running, monitoring can continue even when you are not chatting."),
 ("community","Look beyond the headlines","Compare the bullish and bearish arguments side by side. And remember, community sentiment is an opinion, not a verified company disclosure."),
 ("frameworks","Learn how great investors think","Step into educational frameworks inspired by Buffett, Munger, and Graham. Different perspectives can reveal different questions worth asking."),
 ("overview","InvestOne. A smarter research journey.","From your first question to a deeper investment thesis, InvestOne helps you explore with more clarity. This is a demonstration, using fictional market data.")
]
MODEL_DIR = ROOT / "models"
VOICE = "af_heart"
SPEECH_SPEED = 0.94
# Computed from real synthesized speech; do not truncate a sentence.
SCENE_DURATIONS = []

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
        html = html.replace('<link rel="stylesheet" href="site.css?v=okx-white-v2">', "<style>"+css+"</style>")
        html = html.replace('<script src="site.js?v=okx-white-v2"></script>', "<script>"+js+"</script>")
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
    """Synthesize warm, natural neural speech and pad each scene to its exact length.

    Kokoro-82M (Apache-2.0 model) is local: no third-party API key or
    spoken user data is sent to an external inference service.
    """
    import soundfile as sf
    from kokoro_onnx import Kokoro

    model = MODEL_DIR / "kokoro-v1.0.onnx"
    voices = MODEL_DIR / "voices-v1.0.bin"
    if not model.is_file() or not voices.is_file():
        raise RuntimeError("Missing Kokoro model or voice embeddings")
    tts = Kokoro(str(model), str(voices))
    raw_audio = []
    try:
        for i, (_, _, words) in enumerate(SCENES):
            raw = folder / f"speech_{i:02}.wav"
            wave, sample_rate = tts.create(words, voice=VOICE, speed=SPEECH_SPEED, lang="en-us")
            sf.write(str(raw), wave, sample_rate, subtype="PCM_16")
            length = sf.info(str(raw)).duration
            if length < 0.5:
                raise RuntimeError(f"Kokoro produced empty audio for scene {i}")
            duration = max(6.0, round(length + 1.15, 2))
            SCENE_DURATIONS.append(duration)
            raw_audio.append(raw)
            print(f"Neural voice {i+1}/{len(SCENES)}: {length:.2f}s, scene {duration:.2f}s",flush=True)
    finally:
        del tts

    audio_clips = []
    for i,raw in enumerate(raw_audio):
        padded=folder/f"speech_{i:02}_pad.wav"
        dur=SCENE_DURATIONS[i]
        run(["ffmpeg","-y","-hide_banner","-loglevel","error","-i",str(raw),
             "-af",f"apad=whole_dur={dur},atrim=0:{dur},aresample=44100",
             "-ar","44100","-ac","1",str(padded)])
        audio_clips.append(padded)
    playlist=folder/"audio.ffconcat"
    playlist.write_text("ffconcat version 1.0\n"+"".join(f"file '{x.as_posix()}'\n" for x in audio_clips),encoding="utf-8")
    output=folder/"narration.wav"
    run(["ffmpeg","-y","-hide_banner","-loglevel","error","-safe","0","-f","concat","-i",str(playlist),"-c:a","pcm_s16le",str(output)])
    return output

def create_video(folder):
    clips=[]
    for i in range(len(SCENES)):
        duration=SCENE_DURATIONS[i]
        image=folder/f"frame_{i:02}.png";clip=folder/f"clip_{i:02}.mp4"
        run(["ffmpeg","-y","-hide_banner","-loglevel","error",
             "-threads","1","-filter_threads","1",
             "-loop","1","-framerate","24","-t",str(duration),"-i",str(image),
             "-vf","pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p","-c:v","libx264","-preset","ultrafast",
             "-crf","28","-pix_fmt","yuv420p","-r","24","-an",str(clip)])
        print(f"Encoded scene {i+1}/{len(SCENES)} ({duration:.2f}s)",flush=True)
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
    if types!={"audio","video"} or not 55<duration<180 or path.stat().st_size<100000:
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
        print(f"VOICE={VOICE} · neural Kokoro · scenes={len(SCENES)} · duration={sum(SCENE_DURATIONS):.1f}s",flush=True)

if __name__=="__main__":
    asyncio.run(main())
