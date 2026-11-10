# Live external data verification — independent GitHub runner

**Test:** [Verify Live External Financial Data, run 42665287680](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/actions/runs/42665287680)

**Checked:** 2026-11-09 17:31 UTC, GitHub-hosted Ubuntu runner  
**Code:** `src/live_data_integration_test.go`, using the actual `newLiveProvider()` implementation in `src/data.go` and `src/backup.go`  
**Network:** Real outgoing HTTPS requests. No fixture, hard-coded quote, fake RSS or pre-recorded HTTP response.

| Test | Provider actually used | Recorded outcome |
|---|---|---|
| `SPY` historical prices | Yahoo Finance public chart API | PASS; 22 daily bars; returned quote 752.50 USD, quote timestamp `2026-11-09T17:30:58Z` |
| `NVDA` historical prices | Yahoo Finance public chart API | PASS; 22 daily bars; returned quote 177.92 USD, quote timestamp `2026-11-09T17:30:58Z` |
| `600519.SS` A-share prices | Yahoo Finance public chart API | PASS; 21 daily bars; returned quote 1444.99 CNY, quote timestamp `2026-11-09T07:00:02Z` |
| `NVDA` financial news | Yahoo Finance RSS | PASS; 17 items and a source-date/URL |
| Public investor discussions | Reddit RSS | PASS; 17 items |

These are historical **test-observed** reference prices, not current quotes. The purpose of keeping this record is to demonstrate that the live connection returned actual source-attributed information. It is not a recurring dataset distributed by the repository.

### What is not yet verified

- **SEC EDGAR filings and XBRL:** The app supports these endpoints; the test did not call them because `SEC_CONTACT_EMAIL` was not provided. SEC requests require a legitimate contact email, set in the local app or configured as a GitHub Actions repository secret for testing.
- **Paid ENT207 DeepSeek AI:** Requires a valid user-owned API key. This was not supplied to the GitHub runner; only the local code and mock-based tests were exercised.
- **Mac hardware:** A `darwin/arm64` executable is built in CI, but only running it on a physical Mac verifies network/proxy behavior there.
- **Alternative sources:** Yahoo was available for the tested instruments, so Stooq/Tencent/Google fallback adapters were not exercised by this specific run. Their parsers have separate local tests. Provider availability can change.

### Repeat the verification

Go to [Actions → Verify Live External Financial Data](https://github.com/changxuanyu26-collab/InvestOne_ENT207-/actions/workflows/live-data-probe.yml) and select **Run workflow**; the test also runs on weekdays. A failed result is shown as an error rather than filled with synthetic numbers.

The **GitHub Pages** interface intentionally uses labeled example data; it is a static demonstration, not the real-data Mac backend.
