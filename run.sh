#!/usr/bin/env bash
set -euo pipefail

BACKEND_PORT=8080
FRONTEND_PORT=5173
ML_PORT=8000
ML_DIR="python_ml"
ML_VENV="$ML_DIR/.venv"

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
RESET='\033[0m'

log()  { echo -e "${BLUE}[run]${RESET} $*"; }
ok()   { echo -e "${GREEN}[run]${RESET} $*"; }
warn() { echo -e "${YELLOW}[run]${RESET} $*"; }
die()  { echo -e "${RED}[run]${RESET} $*" >&2; exit 1; }

# ── dependency checks ─────────────────────────────────────────────────────────
command -v go      &>/dev/null || die "go not found. Install Go from https://go.dev/dl/"
command -v npm     &>/dev/null || die "npm not found. Install Node.js from https://nodejs.org/"
command -v python3 &>/dev/null || die "python3 not found. Install Python 3 from https://www.python.org/"

# ── port conflict check ───────────────────────────────────────────────────────
for port in $BACKEND_PORT $FRONTEND_PORT $ML_PORT; do
  if lsof -i ":$port" -sTCP:LISTEN &>/dev/null; then
    die "Port $port is already in use. Stop the conflicting process and retry."
  fi
done

# ── install frontend deps if node_modules is missing ─────────────────────────
if [[ ! -d frontend/node_modules ]]; then
  log "Installing frontend dependencies..."
  npm install --prefix frontend --silent
fi

# ── set up python venv + install ML deps if missing ──────────────────────────
if [[ ! -x "$ML_VENV/bin/python" ]]; then
  log "Creating Python virtualenv for ML service..."
  python3 -m venv "$ML_VENV"
fi

if [[ ! -x "$ML_VENV/bin/uvicorn" ]]; then
  log "Installing ML dependencies (this can take a few minutes on first run)..."
  "$ML_VENV/bin/pip" install --quiet --upgrade pip
  "$ML_VENV/bin/pip" install -r "$ML_DIR/requirements.txt" \
    || die "Failed to install ML dependencies. See errors above."
fi

ML_UVICORN="$(cd "$(dirname "$ML_VENV")" && pwd)/$(basename "$ML_VENV")/bin/uvicorn"

# ── ensure data directory exists ─────────────────────────────────────────────
mkdir -p data

# ── cleanup on exit ───────────────────────────────────────────────────────────
BACKEND_PID=""
FRONTEND_PID=""
ML_PID=""

cleanup() {
  echo ""
  log "Shutting down..."
  [[ -n "$BACKEND_PID"  ]] && kill "$BACKEND_PID"  2>/dev/null || true
  [[ -n "$FRONTEND_PID" ]] && kill "$FRONTEND_PID" 2>/dev/null || true
  [[ -n "$ML_PID"       ]] && kill "$ML_PID"       2>/dev/null || true
  wait "$BACKEND_PID"  2>/dev/null || true
  wait "$FRONTEND_PID" 2>/dev/null || true
  wait "$ML_PID"       2>/dev/null || true
  ok "Stopped."
}
trap cleanup EXIT INT TERM

# ── start ML service ─────────────────────────────────────────────────────────
log "Starting Python ML service..."
(
  cd "$ML_DIR"
  exec "$ML_UVICORN" brain:app --host 127.0.0.1 --port "$ML_PORT" --log-level warning
) &
ML_PID=$!

# Wait for the ML service to accept connections (max 30 s — torch import is slow)
for i in $(seq 1 60); do
  if curl -s "http://127.0.0.1:$ML_PORT/docs" &>/dev/null; then
    break
  fi
  if ! kill -0 "$ML_PID" 2>/dev/null; then
    die "ML service failed to start."
  fi
  sleep 0.5
done

if ! kill -0 "$ML_PID" 2>/dev/null; then
  die "ML service failed to start."
fi

# ── start backend ─────────────────────────────────────────────────────────────
log "Building and starting backend..."
go run ./cmd/server/ &
BACKEND_PID=$!

# Wait for the backend to accept connections (max 10 s)
for i in $(seq 1 20); do
  if curl -s "http://localhost:$BACKEND_PORT/api/readings" &>/dev/null; then
    break
  fi
  sleep 0.5
done

if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
  die "Backend failed to start."
fi

# ── start frontend ────────────────────────────────────────────────────────────
log "Starting frontend..."
npm run dev --prefix frontend --silent &
FRONTEND_PID=$!

sleep 2

# ── ready ─────────────────────────────────────────────────────────────────────
echo ""
ok "Everything is running:"
echo -e "  ${GREEN}Frontend${RESET}   →  http://localhost:$FRONTEND_PORT"
echo -e "  ${GREEN}Backend${RESET}    →  http://localhost:$BACKEND_PORT"
echo -e "  ${GREEN}ML service${RESET} →  http://127.0.0.1:$ML_PORT"
echo -e "  ${GREEN}WebSocket${RESET}  →  ws://localhost:$BACKEND_PORT/ws"
echo ""
echo -e "  API endpoints:"
echo -e "    GET  /api/readings"
echo -e "    GET  /api/anomalies"
echo -e "    POST /api/reports"
echo -e "    GET  /api/forecast?building=library"
echo ""
warn "Press Ctrl+C to stop."
echo ""

# Keep running until a process dies or user hits Ctrl+C
while kill -0 "$BACKEND_PID" 2>/dev/null \
   && kill -0 "$FRONTEND_PID" 2>/dev/null \
   && kill -0 "$ML_PID" 2>/dev/null; do
  sleep 1
done
