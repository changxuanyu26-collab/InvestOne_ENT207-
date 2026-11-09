#!/bin/bash
set -euo pipefail
umask 077
HERE="$(cd "$(dirname "$0")" && pwd)"
BIN="$HERE/InvestOne.app/Contents/MacOS/InvestOne"
APP_SUPPORT="$HOME/Library/Application Support/InvestOne"
PIDFILE="$APP_SUPPORT/investone.pid"
URL="http://127.0.0.1:18791"
if [ "$(uname -s)" != "Darwin" ] || [ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" != "1" ]; then
  echo "InvestOne is for Apple Silicon Macs (M-series)."; exit 1
fi
mkdir -p "$APP_SUPPORT/logs"
if [ ! -f "$BIN" ]; then
  echo "ARM64 application not found. Extract the entire ZIP, then double-click this file."
  exit 1
fi
chmod u+x "$BIN"
EXPECTED_VERSION="$("$BIN" -version 2>/dev/null | /usr/bin/awk '{print $2}')"
HEALTH="$(/usr/bin/curl --noproxy '*' -fsS --max-time 2 "$URL/api/health" 2>/dev/null || true)"
if echo "$HEALTH" | /usr/bin/grep -q '"app":"InvestOne"'; then
  if echo "$HEALTH" | /usr/bin/grep -q "\"version\":\"$EXPECTED_VERSION\""; then
    /usr/bin/open "$URL"
    echo "InvestOne $EXPECTED_VERSION is already running."
    exit 0
  fi
  if [ -f "$PIDFILE" ]; then
    PID="$(/bin/cat "$PIDFILE")"
    if [[ "$PID" =~ ^[0-9]+$ ]]; then
      CMD="$(/bin/ps -p "$PID" -o command= 2>/dev/null || true)"
      if [[ "$CMD" == *'/InvestOne.app/Contents/MacOS/InvestOne'* ]]; then
        /bin/kill -TERM "$PID" 2>/dev/null || true
        for i in {1..40}; do
          if ! /bin/kill -0 "$PID" 2>/dev/null; then break; fi
          /bin/sleep 0.2
        done
      else
        echo "Port 18791 is owned by an unidentified process; none was terminated."
        exit 1
      fi
    fi
  else
    echo "Another InvestOne instance is running without a PID record. Stop it first."
    exit 1
  fi
fi
nohup "$BIN" -open=true > "$APP_SUPPORT/logs/investone.log" 2>&1 < /dev/null &
PID=$!
for i in {1..60}; do
  HEALTH="$(/usr/bin/curl --noproxy '*' -fsS --max-time 1 "$URL/api/health" 2>/dev/null || true)"
  if echo "$HEALTH" | /usr/bin/grep -q "\"version\":\"$EXPECTED_VERSION\""; then
    /usr/bin/open "$URL"
    echo "InvestOne $EXPECTED_VERSION is ready. Close this Terminal window if desired."
    exit 0
  fi
  if ! /bin/kill -0 "$PID" 2>/dev/null; then break; fi
  /bin/sleep 0.25
done
echo "Startup failed. See $APP_SUPPORT/logs/investone.log"
exit 1
