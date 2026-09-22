#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$ROOT_DIR/35sz-api"
LOG_FILE="${LOG_FILE:-$ROOT_DIR/35sz-api.log}"
PID_FILE="${PID_FILE:-$ROOT_DIR/35sz-api.pid}"
PORT="${PORT:-9588}"
REBUILD="${REBUILD:-0}"

build_binary() {
  echo "Building 35sz-api binary..."
  cd "$ROOT_DIR"
  go build -ldflags "-s -w" -o "$BINARY"
}

if [[ -f "$PID_FILE" ]]; then
  EXISTING_PID="$(tr -d '[:space:]' < "$PID_FILE")"
  if [[ -n "$EXISTING_PID" ]] && kill -0 "$EXISTING_PID" 2>/dev/null; then
    echo "35sz-api is already running with PID $EXISTING_PID."
    echo "Run ./stop.sh first if you want to restart it."
    exit 1
  fi
  rm -f "$PID_FILE"
fi

if command -v lsof >/dev/null 2>&1 && lsof -ti tcp:"$PORT" >/dev/null 2>&1; then
  echo "Port $PORT is already in use. Set a different PORT or run ./stop.sh first."
  exit 1
fi

if [[ "$REBUILD" == "1" || ! -x "$BINARY" ]]; then
  build_binary
fi

PORT="$PORT" nohup "$BINARY" > "$LOG_FILE" 2>&1 &
PID="$!"
echo "$PID" > "$PID_FILE"

sleep 0.5
if ! kill -0 "$PID" 2>/dev/null; then
  echo "35sz-api failed to start. Check log file: $LOG_FILE"
  rm -f "$PID_FILE"
  exit 1
fi

echo "Started 35sz-api with PID $PID on port $PORT."
echo "Log file: $LOG_FILE"
