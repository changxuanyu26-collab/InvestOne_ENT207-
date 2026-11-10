<div align="center">

# InvestOne

**AI-powered personal investment intelligence**

*ENT207 · Quantitative Investment and Sector Allocation Strategies in Evolving Capital Markets*

[**▶ Watch product film**](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/demo.mp4) · [**🌐 Try Our App — LIVE**](https://investone-live-production.up.railway.app/) · [**⬇ Download Full App (Windows / Mac / Linux)**](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/) · [**Full app source**](src/)

</div>

## Download InvestOne — six one-click options

<div align="center">

### [⬇ Open the six-platform Download Center](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/)

<a href="https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/InvestOne-v2.1.0-rc1-Windows-x64.zip"><img alt="Download Windows x64" src="https://img.shields.io/badge/⬇_Download-Windows_x64-21D59A?style=for-the-badge&labelColor=101010" /></a>
<a href="https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/InvestOne-v2.1.0-rc1-Windows-ARM64.zip"><img alt="Download Windows ARM64" src="https://img.shields.io/badge/⬇_Download-Windows_ARM64-21D59A?style=for-the-badge&labelColor=101010" /></a>

<a href="https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/InvestOne-v2.1.0-rc1-Mac-AppleSilicon.zip"><img alt="Download Mac Apple Silicon" src="https://img.shields.io/badge/⬇_Download-Mac_Apple_Silicon-21D59A?style=for-the-badge&labelColor=101010" /></a>
<a href="https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/InvestOne-v2.1.0-rc1-Mac-Intel.zip"><img alt="Download Mac Intel" src="https://img.shields.io/badge/⬇_Download-Mac_Intel-21D59A?style=for-the-badge&labelColor=101010" /></a>

<a href="https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/InvestOne-v2.1.0-rc1-Linux-x64.zip"><img alt="Download Linux x64" src="https://img.shields.io/badge/⬇_Download-Linux_x64-21D59A?style=for-the-badge&labelColor=101010" /></a>
<a href="https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/InvestOne-v2.1.0-rc1-Linux-ARM64.zip"><img alt="Download Linux ARM64" src="https://img.shields.io/badge/⬇_Download-Linux_ARM64-21D59A?style=for-the-badge&labelColor=101010" /></a>

</div>

Each button links **directly to its own ZIP**, with no combined archive and no GitHub sign-in required. After downloading, extract it, then double-click the included **Open InvestOne** launcher.

**For Intel / AMD Windows laptops**, including the team's i7-14650HX computer, choose **Windows x64**. For Apple M-series Macs choose **Mac Apple Silicon**.

The six files are published by [GitHub Pages automated build](.github/workflows/pages.yml), with integrity checksums at [SHA256SUMS.txt](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/downloads/SHA256SUMS.txt). [Full platform and installation guide](docs/MULTIPLATFORM.md).

**Signing status:** current Mac builds are unsigned/unnotarized, and Windows binaries are unsigned. Their operating systems may require approval before first launch. A download button does not eliminate these security prompts; a signed/notarized release requires the appropriate developer certificates.

## Live web app — no download, real data

**[Open the InvestOne Landing Page, then click TRY OUR APP](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/)**

The landing page leads to the real Go backend hosted on Railway, not to the old simulated preview. Visitors can access provider-backed financial data without registration. Their research preferences, holdings, alerts and documents are isolated in persistent browser workspaces. For actual AI analysis, visitors enter their own ENT207 DeepSeek API key in the app settings; keys remain in server memory.

The five connected features are real AI Investment Chatbot (user-supplied key), Personalised Intelligence (saved user context), Continuous Automation (server-side rules), Community Integration (real public RSS where available), and Investor Digital Humans (educational Buffett/Munger/Graham AI frameworks, stylized avatars and browser speech synthesis). The last feature is not a clone of any actual investor.

**Independent cloud test:** [Successful GitHub Actions run](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/actions/runs/42684363153) validated HTTPS, visitor isolation, actual SPY quote/source/timestamps from Yahoo, 19 genuine NVDA news items, and safe rejection when no paid AI key was provided. A successful paid model response still needs testing with a legitimate user Key. Providers can rate-limit or become unavailable.

The [original video](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/demo.mp4) remains available, and the [old synthetic interactive preview](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/app/) remains for archival demonstration only.

## Deployment modes

| Edition | Runs where | Data and AI |
|---|---|---|
| **Web App** | Railway HTTPS, reachable from GitHub landing page | Live upstream data, isolated browser workspaces, server-managed shared ENT207 DeepSeek AI, and chat that resets on reload |
| **Desktop** | Mac, Windows and Linux local Go server | Same external providers and user-owned credentials |
| **Archived preview** | GitHub Pages /app/ | Synthetic examples only |

## ENT207TC school DeepSeek model

InvestOne now uses the course gateway **DeepSeek-V4-Flash** (case-sensitive), with server-side endpoint `https://apim-ent207tc.azure-api.net/deepseek/openai/v1`. Calls require the `api-key` header, as well as an OpenAI-compatible bearer credential. The public web app uses one shared course key stored only in the Railway service configuration. Visitors do not enter a key, and its value is never committed to GitHub or embedded in browser code. Public AI use is rate limited. Web chat is kept only in the current page and is cleared when the page is refreshed or reopened; earlier stored web chat histories are purged at service startup.

The course documentation describes a shared budget of **10,000 tokens per minute per group and 5,000,000 tokens per month**. The public AI output cap is 240 tokens to reduce latency and quota use. The local desktop edition keeps its own user-supplied key and conversation history.

In **Investor Digital Humans**, selecting Buffett-, Munger- or Graham-inspired analysis stays on the same page and opens a dedicated AI conversation. The talking avatars are original educational visuals with browser speech synthesis, not real-person likenesses or cloned voices.

## Original five connected core functions

**AI Investment Chatbot** provides a single natural-language research entry point, retrieving available market, filings, financial fact, news, portfolio and local-document context before calling the configured model.

**Personalised Investment Intelligence** stores a research profile, holdings and personal preferences locally to provide relevant context, rather than generic market answers.

**Continuous Monitoring & Automation** evaluates user rules and prepares alerts or refreshed analysis while the Mac service is running and awake.

**Forum & Community Integration** distinguishes unverified public discussion from dated financial sources, and supports saved evidence-backed opinions. Access depends on the external forum.

**Investor Digital Humans** offers Buffett-, Munger- and Graham-inspired *educational analysis frameworks* using publicly documented concepts and the LLM. Not an endorsement, a real-person clone, or financial advice.

The backend also includes a market research panel, SEC fact and filings viewer, user research library, paper trades, portfolio context, factor analysis, sector allocation and historical multi-factor backtesting.

## Real financial data: implemented and independently tested

The Go source under `src/` uses actual HTTPS provider responses and records provider attribution, quote/reference time, retrieval time, source link and adjustment flags. The frontend displays errors instead of filling gaps with invented live prices.

| Research function | Current provider path | Notes |
|---|---|---|
| US equity and ETF quotes + history | Yahoo Finance chart 1/2 → Stooq daily prices | Unofficial and may be delayed; Stooq is **not real-time** |
| China A-share daily bars | Yahoo chart 1/2 → Tencent public daily-K | Third-party public endpoints; adjustments differ |
| Financial news | Yahoo Finance RSS → Google News RSS | Headline/snippet results; source dates are shown |
| SEC company filings and financial facts | SEC EDGAR submissions / XBRL companyfacts | Requires a **real SEC contact email**, entered in local Settings |
| Community discussion | Public Reddit RSS | Unavailable if blocked or restricted; local evidence saving remains usable |
| AI analysis | ENT207 DeepSeek OpenAI-compatible API | Shared server key on the public site; user-supplied key in the desktop edition |

**Independent evidence:** The [GitHub live-provider test run](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/actions/runs/42665287680) successfully fetched actual SPY, NVDA and 600519.SS chart data, NVDA news and Reddit public posts on Nov 9, 2026 (UTC). The SEC and paid ENT207 DeepSeek calls were **not tested** in that run: no legitimate SEC contact email or user-paid AI key was supplied. Third-party availability may change; run the app's data-connection diagnostics on your Mac.

See [detailed desktop setup and data-provider guide](docs/DESKTOP_GUIDE.md).

## Legacy Apple Silicon release (V2.0 RC)

1. For the current cross-platform build use the V2.1 downloads above; the old [InvestOne V2 Apple Silicon ZIP](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/tag/v2.0.0-rc2).
2. Unzip it and double-click **Open InvestOne.command**. The bundled app is compiled for Mac M-series; no Python, Node, or Docker installation is required.
3. Sign in with `红豆薏米水` or `RedBeanCoix` and, for AI analysis, enter your ENT207 Azure DeepSeek API Key.
4. Open **Settings → Check live data connections** and inspect each endpoint, its source and timestamp. For official SEC data, supply your own legitimate contact email.

All personal state remains under `~/Library/Application Support/InvestOne/`. No API key or user portfolio is committed to GitHub. The Mac binary is not Apple-signed/notarized; macOS may require explicit approval for an app you trust.

## Source, tests and build

The full, inspectable application is in [`src/`](src/); local entrypoint `src/main.go`, provider adapters `src/data.go` and `src/backup.go`, model integration `src/ai.go`, monitoring `src/monitor.go`, multi-factor backtest `src/quant.go`, and the five-core web interface in `src/web/`.

```bash
cd src
go test ./...
go vet ./...
go run . -open=false
# Explicit external request tests; do not run against fake fixtures:
INVESTONE_LIVE_PROBE=1 go test -count=1 -v -run TestLiveFinancialProviders
```

Automated workflows:
- [Mac ARM64 test/build/release](.github/workflows/mac-release.yml) — publishes the actual one-click ZIP and SHA256 checksum.
- [Live real-market probe](.github/workflows/live-data-probe.yml) — requests authentic third-party feeds and reports failures.
- [GitHub Pages interactive demo](.github/workflows/pages.yml) — preserves the originally published narrated video; the preview remains synthetic.

## Technical and safety notes

- Market providers can be rate-limited, delayed, discontinued or subject to license restrictions. The code performs runtime requests rather than distributing commercial or proprietary historical feeds.
- Backtests use past signals and explicit modeling assumptions, but adjusted prices, fixed universes, execution simulation and selection biases remain research limitations.
- All trades are **paper only**. No real brokerage order placement.
- Monitoring runs while the desktop service is awake; a static GitHub Pages app cannot monitor continuously.
- The local username is not a secure internet login. **Never expose the localhost Go server publicly.**
- Digital-human features are educational prompts, not real investor identities, endorsement, likenesses or voice cloning.
- See [SECURITY.md](SECURITY.md), [ARCHITECTURE.md](ARCHITECTURE.md), [LIMITATIONS.md](LIMITATIONS.md) for full boundaries.

**InvestOne is an educational research prototype and does not provide regulated investment advice.**
