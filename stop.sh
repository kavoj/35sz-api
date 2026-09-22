#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="${PID_FILE:-$ROOT_DIR/35sz-api.pid}"
PORT="${PORT:-9588}"

stop_pid() {
  local pid="$1"
  if [[ -z "$pid" ]] || ! kill -0 "$pid" 2>/dev/null; then
    return 0
  fi
  echo "Stopping process $pid..."
  kill "$pid" 2>/dev/null || true
  for _ in {1..25}; do
    kill -0 "$pid" 2>/dev/null || return 0
    sleep 0.2
  done
  kill -9 "$pid" 2>/dev/null || true
}

if [[ -f "$PID_FILE" ]]; then
  stop_pid "$(tr -d '[:space:]' < "$PID_FILE")"
  rm -f "$PID_FILE"
fi

if command -v lsof >/dev/null 2>&1; then
  while IFS= read -r pid; do
    stop_pid "$pid"
  done < <(lsof -ti tcp:"$PORT" || true)
fi

echo "Stopped 35sz-api services on port $PORT."
