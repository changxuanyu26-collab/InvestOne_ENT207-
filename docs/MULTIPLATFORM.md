# InvestOne · Download for your computer

InvestOne runs a private Go server on **127.0.0.1:18791** and opens its web interface in your default browser. All six release ZIPs use the same core Go application; no Python, Node.js, CUDA, or Docker is required to use the app.

## Choose the correct download

| Device | Download | Open after extraction |
|---|---|---|
| Mac with Apple Silicon (M1–M series) | **Mac-AppleSilicon** | `Open InvestOne.command` |
| Mac with Intel processor | **Mac-Intel** | `Open InvestOne.command` |
| Windows 10/11, Intel / AMD x64 | **Windows-x64** | `Open InvestOne.cmd` |
| Windows 11 on ARM | **Windows-ARM64** | `Open InvestOne.cmd` |
| Linux, Intel / AMD x86-64 | **Linux-x64** | `Open InvestOne.sh` |
| Linux, ARM64 | **Linux-ARM64** | `Open InvestOne.sh` |

**Example team member:** Intel Core i7-14650HX, 16 GB RAM, GeForce RTX 4060 Laptop, 64-bit Windows on x64: choose **Windows-x64**. No GPU is needed for market data retrieval or calling cloud AI APIs.

## First launch
1. Download the ZIP for your OS and architecture from the GitHub Release assets list.
2. Extract the *whole* ZIP to a normal folder.
3. Double-click the platform's **Open InvestOne** launcher (Linux may require making it executable once). Keep the application files together.
4. The localhost web app opens. Enter **红豆薏米水** (or **RedBeanCoix**) plus **your own Highway API Key**.
5. Open the live data diagnostics to check actual financial providers and timestamps.

The Windows and Linux launchers open the URL `http://127.0.0.1:18791`. If the browser fails to launch automatically, enter that address manually.

## Data locations
- Mac: `~/Library/Application Support/InvestOne/`
- Windows: `%APPDATA%\InvestOne\`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/InvestOne/`

API keys are held in memory during the local session. Never commit or share them. Login alias is a **local convenience** rather than secure public-web authentication. Do **not** expose port 18791 to the internet.

## Platform protections
The binaries are presently **not code signed or notarized**. Operating systems may display security warnings, especially on the first launch. Run only if you trust the published GitHub release and its checksum. Release ZIP checksums are in `SHA256SUMS.txt` on the same Release page.

## Important capability boundaries
- The public GitHub Pages interactive preview is a synthetic demonstration.
- The downloadable application calls real upstream financial and news sources at runtime; requests can be throttled, blocked or unavailable. A successful GitHub Actions smoke test does not guarantee connectivity on every device or network.
- Real AI requires your Highway API Key.
- SEC regulatory data requires a legitimate contact email entered in Settings.
- Continuous monitoring runs while the local service is running and the computer is awake.
- Investor-framework conversations are educational analyses, not endorsements or real-person voice/face clones.
- No actual broker orders. Paper trading only.

## Build from source
Source: `src/`, entrypoint `src/main.go`. Release workflow: `.github/workflows/mac-release.yml` (name retained for backwards links; it builds every platform).

```bash
cd src
go test ./...
go vet ./...
# Example Windows x64 binary:
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o InvestOne.exe .
```
