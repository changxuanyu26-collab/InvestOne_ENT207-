<div align="center">

# InvestOne

**AI-Powered Personal Investment Intelligence**

An educational investment research workspace that combines market evidence, simulated portfolios, quantitative backtesting, monitoring, a local research library, and AI-assisted analysis.

**ENT207 · Red Bean & Coix · Mac Apple Silicon / Browser Demo**

[Explore the interactive demo](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/) · [Demo source](docs/index.html) · [Architecture](ARCHITECTURE.md) · [Safety & data notes](SECURITY.md)

</div>

> **Demo data notice** — The public browser demo is an interactive **offline simulation**. The sample market prices, events, financial statistics and any scripted "AI" response are illustrative, not live financial data, not live model responses, and not investment advice. The separately distributed local desktop app connects to third-party financial endpoints and can call a user-configured Highway API, with availability dependent on the providers.

## Product overview

| Workspace | What it does |
|---|---|
| Overview | Summary of paper assets, watchlist, monitored events, and research actions |
| Market research | Tickers, historical charts and sourced financial research |
| Portfolio | Virtual cash, positions, paper trades and approximate portfolio risk |
| Quant Lab | Monthly sector allocation research, factor weighting and historical backtests |
| AI Assistant | Evidence-aware research with the user's Highway API key (desktop app only) |
| Research Library | Locally saved documents and analysis context |
| Automation | User-defined market monitoring rules while the desktop service runs |
| Community Insights | Separates public sentiment and opinions from official disclosures |
| Investment Frameworks | Educational simulations inspired by publicly documented Buffett, Munger and Graham principles |

## Demo

The public demo in `docs/index.html` is a static, dependency-free website intended for GitHub Pages. It demonstrates page navigation, watchlist interaction, a paper trading example, interactive research, rule creation, a quant illustration, and a scripted AI walkthrough. All presented market data is **synthetic** and clearly labeled.

The demo never requests or stores API keys. The desktop product must never be represented as providing reliable real-time market data without external provider verification.

### Publish on GitHub Pages

Repository owner: **Settings → Pages → Build and deployment → Deploy from a branch → main → /docs → Save**. After GitHub finishes deployment, the demo will be at:

https://changxuanyu26-collab.github.io/InvestOne_ENT207-/

The link may return 404 until GitHub Pages is enabled.

## Desktop application

The private/local desktop distribution is a **macOS Apple Silicon** application with a Go HTTP server bound to localhost and an embedded browser interface. It is not a multi-user authenticated cloud service.

- Default login alias: `RedBeanCoix`
- Legacy Chinese alias remains supported for existing users
- AI provider: Highway OpenAI-compatible API
- Default model: `deepseek/deepseek-v4.1-flash`
- API key supplied at login and held only in process memory
- Paper trades use virtual money; no brokerage orders
- Data providers are third-party services and may be unavailable, delayed or restricted

See [Architecture](ARCHITECTURE.md), [Security](SECURITY.md), and [limitations](LIMITATIONS.md).

## Engineering principles

**Source provenance** · **No invented live prices** · **Paper trading only** · **Data minimization** · **Look-ahead-safe backtest timing** · **Explicit limitations**.

The public materials are coursework demonstration artifacts, not investment recommendations, a broker, or a production-grade financial advisory system.

## Development status

- English-language UI and local Go app: developer build and automated tests.
- Public interactive demo: standalone front-end with simulated data.
- Third-party live data reliability: **not verified end-to-end on the user's Mac**.
- Highway AI calls: **not verified with a paid key in the test environment**.
- Public repo: will be expanded with audited source, tests and artifacts; file presence must not be inferred from the local distribution.

## Technology

Go 1.23 standard library · HTML/CSS/JavaScript · SVG charts · GitHub Pages (demo) · macOS ARM64 (local desktop app).

---
**Privacy:** never commit a real API key, locally saved portfolio state, contact email, personally identifying file path or unlicensed third-party data.
