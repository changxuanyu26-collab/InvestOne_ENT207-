#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$ROOT/InvestOne"
URL="http://127.0.0.1:18791"
if [[ ! -f "$BIN" ]]; then
 echo "InvestOne binary missing. Extract the entire ZIP and retry."
 exit 1
fi
chmod u+x "$BIN"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/InvestOne"
mkdir -p "$DATA_DIR/logs"
nohup "$BIN" -open=true > "$DATA_DIR/logs/investone.log" 2>&1 < /dev/null &
PID=$!
echo "Starting InvestOne (PID $PID)..."
for attempt in {1..40}; do
 if command -v curl >/dev/null 2>&1 && curl --noproxy '*' -fsS --max-time 1 "$URL/api/health" >/dev/null 2>&1; then
  if command -v xdg-open >/dev/null 2>&1; then xdg-open "$URL" >/dev/null 2>&1 || true; fi
  echo "InvestOne is available at $URL"
  exit 0
 fi
 if ! kill -0 "$PID" 2>/dev/null; then
  echo "InvestOne did not start. Check $DATA_DIR/logs/investone.log"
  exit 1
 fi
 sleep .25
done
echo "Please open $URL in your browser. Log: $DATA_DIR/logs/investone.log"
