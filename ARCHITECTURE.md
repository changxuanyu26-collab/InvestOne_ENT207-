# InvestOne architecture

## Local application
The full local application is written in Go 1.23 with an embedded HTML/CSS/JavaScript interface. It binds only to `127.0.0.1:18791`; it is not an internet-facing multi-user platform.

```text
Browser UI (local)
  -> Go HTTP API + session cookie
     -> Private on-disk JSON store: profile, watchlist, simulated trades, chat, documents, rules
     -> Market provider adapters: Yahoo Finance, fallback daily-price services
     -> SEC filings and financial fact retrieval where supported
     -> News RSS and public community feed retrieval where available
     -> Quant Lab: monthly multi-factor strategy, execution lag and fees
     -> AI: Highway OpenAI-compatible Chat Completions
     -> Background monitoring while the application is running
```

### AI
- Default OpenAI-compatible base URL: `https://api.highwayapi.ai/openai`
- Default model: `deepseek/deepseek-v4.1-flash`
- Key is supplied by the local user and kept in server process memory only.
- Citations are restricted to retrieved evidence and are not guaranteed to establish every claim.
- Third-party terms, quota and data availability apply.

### Quantitative analysis
The local research engine forms an equal-weight or scored, monthly rebalanced portfolio using cross-sectional momentum, trend and volatility components. Controls include top-N holdings, a trend-based cash switch and transaction fees. Signal information is lagged relative to execution.

**Limitations:** historical adjusted-price quality depends on the upstream feed; fixed-universe selection creates survivorship bias; backtests are educational simulations rather than forecasts; no real-time brokerage execution.

## Public demo
`docs/index.html` is a **self-contained offline interactive browser demo**, deployed separately from the full app. It does not authenticate or contact APIs. All displayed data are synthetic illustrative examples. The quant chart is not a backtest, and the AI reply is scripted.

## Running locally
The macOS release bundle contains an Apple Silicon ARM64 executable and one-click shell launcher. It preserves local research data under `~/Library/Application Support/InvestOne/`. A developer build uses `cd src && go test ./... && go build .`.

**Do not run the server on a public interface; its sign-in alias is not a security boundary for internet access.**
