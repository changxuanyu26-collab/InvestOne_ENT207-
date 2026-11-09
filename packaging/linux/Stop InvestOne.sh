#!/usr/bin/env bash
set -euo pipefail
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/InvestOne"
PIDFILE="$DATA_DIR/investone.pid"
if [[ ! -f "$PIDFILE" ]]; then echo "No running InvestOne process recorded."; exit 0; fi
PID="$(cat "$PIDFILE")"
if [[ ! "$PID" =~ ^[0-9]+$ ]]; then echo "Invalid process ID"; exit 1; fi
CMD="$(tr '\0' ' ' < "/proc/$PID/cmdline" 2>/dev/null || true)"
if [[ "$CMD" == *"/InvestOne"* ]]; then kill -TERM "$PID"; echo "InvestOne stopped."; else echo "Unrelated or inactive process preserved."; fi
