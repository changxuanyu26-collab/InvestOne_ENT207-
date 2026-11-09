<div align="center">

# InvestOne

**AI-powered personal investment intelligence**

*ENT207 · Quantitative Investment and Sector Allocation Strategies in Evolving Capital Markets*

[**▶ Watch product film**](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/?autoplay=1) · [**Try five-core interactive preview**](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/) · [**⬇ Download Full App (Windows / Mac / Linux)**](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/tag/v2.1.0-rc1) · [**Full app source**](src/)

</div>

## Download the actual application

**[⬇ Download InvestOne for your computer — Windows / macOS / Linux](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/tag/v2.1.0-rc1)**

| Computer | Recommended downloadable ZIP |
|---|---|
| **Windows Intel / AMD (x64)** | [**Download Windows x64**](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/download/v2.1.0-rc1/InvestOne-v2.1.0-rc1-Windows-x64.zip) |
| Windows ARM (ARM64) | [Download Windows ARM64](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/download/v2.1.0-rc1/InvestOne-v2.1.0-rc1-Windows-ARM64.zip) |
| **Mac with Apple M chip** | [**Download macOS Apple Silicon**](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/download/v2.1.0-rc1/InvestOne-v2.1.0-rc1-Mac-AppleSilicon.zip) |
| Mac with Intel processor | [Download macOS Intel](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/download/v2.1.0-rc1/InvestOne-v2.1.0-rc1-Mac-Intel.zip) |
| Linux Intel / AMD (x64) | [Download Linux x64](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/download/v2.1.0-rc1/InvestOne-v2.1.0-rc1-Linux-x64.zip) |
| Linux ARM64 | [Download Linux ARM64](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/download/v2.1.0-rc1/InvestOne-v2.1.0-rc1-Linux-ARM64.zip) |

Extract the ZIP and double-click `Open InvestOne.cmd` on Windows or `Open InvestOne.command` on Mac. Linux users launch `Open InvestOne.sh` (some desktop environments require allowing execution first). This runs the complete local Go backend and automatically opens `http://127.0.0.1:18791`, without Python, Node.js or Docker. Log in as **红豆薏米水** and enter your own Highway API key.

**For the group member with Intel Core i7-14650HX + RTX 4060 Laptop + 64-bit Windows x64: choose Windows x64.** No NVIDIA GPU setup is needed to access external financial data and hosted AI. The download remains architecture-specific; Windows ARM64 is for ARM PCs, not Intel CPUs.

Read the [complete multi-platform instructions](docs/MULTIPLATFORM.md). The application bundles provider adapters for real external financial feeds but depends on each provider's uptime, network access and license conditions. The desktop binaries are not code signed.

## Two deliverables — clearly distinguished

| Product | Deployment | Data / AI |
|---|---|---|
| **Full InvestOne V2 Desktop App** | Apple Silicon Mac, Go local backend; release ZIP | Requests real market, news, forum and SEC endpoints at runtime; user-provided Highway API key for LLM |
| **GitHub Pages interactive preview** | Public static site in `docs/` | Explicitly synthetic financial numbers and scripted AI; no backend, secret keys or brokerage access |

The static GitHub Pages website **cannot run the Go desktop backend or safely hold a private model API key**. It remains a visual/interactive preview. To use external financial feeds and the real AI workflow, launch the Mac app locally.

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
| AI analysis | User-supplied Highway OpenAI-compatible API | Default `deepseek/deepseek-v4.1-flash`; actual key required |

**Independent evidence:** The [GitHub live-provider test run](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/actions/runs/42665287680) successfully fetched actual SPY, NVDA and 600519.SS chart data, NVDA news and Reddit public posts on Nov 9, 2026 (UTC). The SEC and paid Highway calls were **not tested** in that run: no legitimate SEC contact email or user-paid AI key was supplied. Third-party availability may change; run the app's data-connection diagnostics on your Mac.

See [detailed desktop setup and data-provider guide](docs/DESKTOP_GUIDE.md).

## Legacy Apple Silicon release (V2.0 RC)

1. For the current cross-platform build use the V2.1 downloads above; the old [InvestOne V2 Apple Silicon ZIP](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/releases/tag/v2.0.0-rc2).
2. Unzip it and double-click **Open InvestOne.command**. The bundled app is compiled for Mac M-series; no Python, Node, or Docker installation is required.
3. Sign in with `红豆薏米水` or `RedBeanCoix` and, for AI analysis, enter your Highway API Key.
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
