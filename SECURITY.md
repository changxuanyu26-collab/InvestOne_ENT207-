# Security & privacy

## Public repository rules
Never upload real API keys, environment files, account identifiers, portfolio exports, private research documents, `~/Library/Application Support/InvestOne` state, or user-specific screenshots.

## Model API
The local desktop application holds the configured Highway API key in process memory and does not write it into the local JSON store. Requests send the question and selected source/profile context to the configured provider. The external provider may log or process requests under its own policies. Avoid adding confidential material to prompts.

## Local access model
The desktop app is **localhost-only, single-user coursework software**, not a hardened cloud authentication solution. Its username is a convenience alias, not a password. Do not expose its Go HTTP port publicly.

## Market data
Market and news endpoints are unofficial or subject to third-party terms, outages, rate limits, and changing formats. An external data source must not be represented as fully verified merely because parsing or synthetic tests succeeded. Failed requests must remain visible instead of being filled with invented prices.

## Public demo
No API keys, cookies, auth, live data fetch, financial transactions, or database are used by the static GitHub Pages demonstration. All showcased data are synthetic. The example AI answer is scripted.

## Responsible use
Paper trading only. No broker integration, no personalized financial advice, and no claim that simulated investor frameworks represent or are endorsed by real people.
