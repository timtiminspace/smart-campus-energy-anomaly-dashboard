#!/usr/bin/env bash
# Surrey VM bootstrap — Phase 6 deploy for team-hendrixx.
#
# Target: Ubuntu 24.04 VM at user@10.2.8.118 (SSH from Heron/Otter only).
# Public URL: https://com2042-hendrixx.csee.surrey.ac.uk (gateway → :3000).
#
# Idempotent: safe to re-run. Each step skips if already satisfied.
#
# Usage (from a Heron/Otter machine):
#   scp deploy/vm-bootstrap.sh user@10.2.8.118:~/
#   ssh user@10.2.8.118
#   GITLAB_PAT=glpat-xxxxxxxx ./vm-bootstrap.sh
#
# Required env:
#   GITLAB_PAT   Surrey GitLab Personal Access Token (scope: read_repository)
#
# Optional env:
#   REPO_URL     defaults to the team-hendrixx HTTPS URL
#   REPO_DIR     defaults to ~/team-hendrixx
#   GIT_BRANCH   defaults to main

set -euo pipefail

REPO_URL="${REPO_URL:-https://gitlab.surrey.ac.uk/csee/com2042/2025-26-groups/team-hendrixx.git}"
REPO_DIR="${REPO_DIR:-$HOME/team-hendrixx}"
GIT_BRANCH="${GIT_BRANCH:-main}"

log() { printf '\n\033[1;36m▶ %s\033[0m\n' "$*"; }

# ── 1. Docker Engine + Compose plugin (task 24) ─────────────────────────────
if ! command -v docker >/dev/null 2>&1; then
    log "Installing Docker Engine + Compose plugin"
    sudo apt-get update
    sudo apt-get install -y ca-certificates curl gnupg
    sudo install -m 0755 -d /etc/apt/keyrings
    if [ ! -f /etc/apt/keyrings/docker.gpg ]; then
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
            | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        sudo chmod a+r /etc/apt/keyrings/docker.gpg
    fi
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu noble stable" \
        | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
    sudo apt-get update
    sudo apt-get install -y \
        docker-ce docker-ce-cli containerd.io \
        docker-buildx-plugin docker-compose-plugin
    sudo usermod -aG docker "$USER"
    log "Added $USER to docker group — log out/in (or run \`newgrp docker\`) for it to take effect"
else
    log "Docker already installed: $(docker --version)"
fi

# Ensure Docker comes back on reboot (task 28).
sudo systemctl enable --now docker
log "Docker systemd unit: $(systemctl is-enabled docker) / $(systemctl is-active docker)"

# ── 2. Clone (or update) the repo (task 25) ─────────────────────────────────
if [ -z "${GITLAB_PAT:-}" ] && [ ! -d "$REPO_DIR/.git" ]; then
    echo "ERROR: GITLAB_PAT must be set on first run (read_repository scope)" >&2
    exit 1
fi

if [ -d "$REPO_DIR/.git" ]; then
    log "Updating existing checkout at $REPO_DIR"
    git -C "$REPO_DIR" fetch --all --prune
    git -C "$REPO_DIR" checkout "$GIT_BRANCH"
    git -C "$REPO_DIR" pull --ff-only origin "$GIT_BRANCH"
else
    log "Cloning $REPO_URL → $REPO_DIR"
    # Embed the PAT only in the clone command, then strip it from the stored
    # remote URL so it never lands on disk under .git/config.
    AUTH_URL="${REPO_URL/https:\/\//https://oauth2:${GITLAB_PAT}@}"
    git clone --branch "$GIT_BRANCH" "$AUTH_URL" "$REPO_DIR"
    git -C "$REPO_DIR" remote set-url origin "$REPO_URL"
fi

# Pull subsequent updates over HTTPS without re-prompting for the PAT. This
# stores the credential in ~/.git-credentials with 0600 perms — acceptable on a
# single-user VM, but rotate the PAT if the VM is ever shared.
if [ -n "${GITLAB_PAT:-}" ]; then
    git config --global credential.helper store
    umask 077
    printf 'https://oauth2:%s@gitlab.surrey.ac.uk\n' "$GITLAB_PAT" > "$HOME/.git-credentials"
fi

# ── 3. Bring the stack up (task 25) ─────────────────────────────────────────
cd "$REPO_DIR"
log "docker compose up -d --build"
# Use sudo if the docker group hasn't taken effect in this shell yet.
if docker info >/dev/null 2>&1; then
    DOCKER="docker"
else
    DOCKER="sudo docker"
fi
$DOCKER compose up -d --build
$DOCKER compose ps

# ── 4. Smoke-test the same-origin gateway path (task 26) ────────────────────
log "Smoke-testing http://127.0.0.1:3000"
sleep 5
curl -sf -o /dev/null -w "  GET /            → HTTP %{http_code}\n" http://127.0.0.1:3000/
curl -sf -o /dev/null -w "  GET /api/anomalies → HTTP %{http_code}\n" http://127.0.0.1:3000/api/anomalies
curl -sf -o /dev/null -w "  GET /api/readings  → HTTP %{http_code}\n" 'http://127.0.0.1:3000/api/readings?limit=1'
curl -sf -o /dev/null -w "  GET /report (SPA)  → HTTP %{http_code}\n" http://127.0.0.1:3000/report
WS_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Connection: Upgrade" -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 13" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    --max-time 3 http://127.0.0.1:3000/ws || true)
echo "  GET /ws (upgrade)  → HTTP $WS_CODE  (101 = success)"

log "Done. From a Heron/Otter machine, hit:"
echo "    https://com2042-hendrixx.csee.surrey.ac.uk"
