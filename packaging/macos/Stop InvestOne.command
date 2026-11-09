#!/bin/bash
set -euo pipefail
PIDFILE="$HOME/Library/Application Support/InvestOne/investone.pid"
if [ ! -f "$PIDFILE" ]; then echo "No running InvestOne instance recorded."; exit 0; fi
PID="$(cat "$PIDFILE")"
case "$PID" in ''|*[!0-9]*) echo "Invalid PID. Nothing stopped."; exit 1;; esac
CMD="$(ps -p "$PID" -o command= 2>/dev/null || true)"
case "$CMD" in
  *InvestOne.app/Contents/MacOS/InvestOne*) kill -TERM "$PID"; echo "InvestOne stopped; the in-memory AI key has been cleared.";;
  *) echo "No matching InvestOne process. Unrelated process preserved.";;
esac
