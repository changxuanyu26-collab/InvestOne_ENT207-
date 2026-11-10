# InvestOne V2 — Mac Apple Silicon

**Complete local application, real-data endpoints and five integrated functions.**

The GitHub Pages site is a separate educational preview with sample data. This ZIP contains the **real backend** and must run on an Apple Silicon Mac (M-series) to use external providers and optional AI.

## One-click start

1. Download the latest **InvestOne-V2-Mac-AppleSilicon.zip** from the GitHub Release.
2. Unzip the whole archive, retaining the `InvestOne.app` folder and the launcher.
3. On your Mac, double-click `Open InvestOne.command`. A browser tab opens at `http://127.0.0.1:18791`.
4. Sign in as `红豆薏米水` or `RedBeanCoix`, and enter your ENT207 DeepSeek API key to use the AI Assistant. The Key is held only in process memory and is not committed, persisted or shared with GitHub.
5. In Settings, press **Check live data connections**. Verify the live data provider, reference timestamps and connection status. If a feed fails, its actual error appears; the program does not invent substitute quotes.
6. For U.S. SEC filings and XBRL facts, enter your real contact email in Settings. This is required for a compliant SEC User-Agent, independently of the AI API key.
7. Use `Stop InvestOne.command` to stop the local background service. In-memory AI credentials are cleared, and local research data remains on your Mac.

If macOS flags the unsigned app as untrusted, only approve it if you obtained the ZIP from this verified repository. The project is not Apple-notarized; it is coursework software.

## What is implemented

- **AI Investment Chatbot:** OpenAI-compatible ENT207 DeepSeek Chat Completions, documented fact/news/portfolio context, citations to acquired source IDs and persisted history.
- **Personalised Intelligence:** locally stored profile, watchlist, investment horizon and research preferences; paper portfolio context.
- **Continuous Monitoring:** configured price/earnings/news/periodic rules, checks and notifications while the desktop service stays awake.
- **Community Integration:** Reddit RSS where publicly accessible, saved opinions, source/author labels; availability is provider-dependent.
- **Investor Digital Humans:** educational Buffett, Munger and Graham analytical framing using the same AI API; not a voice/avatar clone or endorsed by actual investors.

Professional tools — market research, SEC fact inspection, document library, monthly multi-factor portfolio backtesting, portfolio metrics and paper trading — are accessible inside the app.

## Real financial data

| Category | Primary source | Fallback or requirement | Limitations |
|---|---|---|---|
| US prices / daily history | Yahoo Finance public chart endpoint | Stooq US daily CSV | Unofficial feeds; delayed or unavailable; Stooq cannot be used as tradable realtime data |
| China A-share daily history | Yahoo public chart endpoint | Tencent public daily K line | Unofficial endpoints and differing adjustments |
| Financial news | Yahoo RSS | Google News RSS | Headline and snippet aggregation, not necessarily complete articles |
| Official US filings | SEC EDGAR | Requires your real contact email | SEC may rate-limit or restrict access |
| SEC financial facts | SEC companyfacts XBRL | Requires your real contact email | Report period and filing date must be respected |
| Public investment forums | Reddit RSS | User-saved sourced opinion | May be blocked or empty; not a comprehensive market sentiment service |

**Data provenance:** prices carry source, reference date/time, retrieved time, adjustment flag, and source URL. Stooq/Tencent daily `AsOf` values are day references, not exact trade times. Real-time quote quality depends on source permissions and availability.

### Independent verification in GitHub

Open **Actions → Verify Live External Financial Data**. The opt-in Go integration test actually requests SPY, NVDA, 600519.SS and NVDA financial news, validating timestamps, source attribution and usable bars. Failed providers are reported as failures and not silently replaced with fake data. SEC tests run when a genuine `SEC_CONTACT_EMAIL` repository secret is available. A GitHub runner's result does not guarantee your Mac's network connection; verify locally as well.

To run locally from source:

```bash
cd src
go test ./...
go build .
INVESTONE_LIVE_PROBE=1 go test -count=1 -v -run TestLiveFinancialProviders
```

No live broker order execution, no hosted multi-user login and no guarantee of uninterrupted third-party access are provided. External-market endpoints should be used in accordance with their terms and licenses. Market data are accessed at runtime: **GitHub does not hold a constantly updating proprietary feed or private API keys**.

## Security

The application binds exclusively to `127.0.0.1`. The username is a local convenience alias and **must not be used as internet-facing authentication**. Do not publicly forward the port. Use the separate GitHub Pages preview for public demonstration.

Paper trading is simulated; nothing is routed to a brokerage. This is educational research software, not personalized professional investment advice. The locally saved portfolio/profile state stays outside the GitHub repo.

Source: [github.com/changxuanyu26-collab/InvestOne_ENT207-](https://github.com/changxuanyu26-collab/InvestOne_ENT207-)
