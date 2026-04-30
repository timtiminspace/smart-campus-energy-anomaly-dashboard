#!/usr/bin/env bash
# Install GitLab Runner on the Surrey VM and register it for team-hendrixx.
#
# Run on the VM AFTER deploy/vm-bootstrap.sh has installed Docker.
#
# How to get a token:
#   GitLab → Project → Settings → CI/CD → Runners → "New project runner"
#   Tags: hendrixx-vm   Run untagged: no   Locked to project: yes
#   GitLab will show a one-shot authentication token (glrt-...).
#
# Usage:
#   RUNNER_TOKEN=glrt-xxxxxxxxxxxxxxxxxxxx ./register-gitlab-runner.sh
#
# Optional env:
#   GITLAB_URL    defaults to https://gitlab.surrey.ac.uk
#   RUNNER_DESC   defaults to "hendrixx-vm"

set -euo pipefail

if [ -z "${RUNNER_TOKEN:-}" ]; then
    echo "ERROR: RUNNER_TOKEN required (Settings → CI/CD → Runners → New project runner)" >&2
    exit 1
fi

GITLAB_URL="${GITLAB_URL:-https://gitlab.surrey.ac.uk}"
RUNNER_DESC="${RUNNER_DESC:-hendrixx-vm}"

log() { printf '\n\033[1;36m▶ %s\033[0m\n' "$*"; }

# ── 1. Install gitlab-runner from the official repo ─────────────────────────
if ! command -v gitlab-runner >/dev/null 2>&1; then
    log "Installing gitlab-runner"
    curl -fsSL "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" \
        | sudo bash
    sudo apt-get install -y gitlab-runner
else
    log "gitlab-runner already installed: $(gitlab-runner --version | head -1)"
fi

# ── 2. Give the runner user docker access ──────────────────────────────────
# The shell executor invokes `docker compose` as the gitlab-runner user, so it
# needs to be in the docker group. Restart docker so the new group sticks.
sudo usermod -aG docker gitlab-runner
sudo systemctl restart docker

# ── 3. Register with the project ────────────────────────────────────────────
# Newer GitLab (≥16.6) uses authentication tokens (glrt-...) with --token.
# If your token is a legacy registration token instead, swap --token for
# --registration-token.
log "Registering runner against $GITLAB_URL"
sudo gitlab-runner register --non-interactive \
    --url "$GITLAB_URL" \
    --token "$RUNNER_TOKEN" \
    --description "$RUNNER_DESC" \
    --executor shell \
    --tag-list "hendrixx-vm"

# ── 4. Auto-start on reboot ────────────────────────────────────────────────
sudo systemctl enable --now gitlab-runner
log "Runner state: $(systemctl is-enabled gitlab-runner) / $(systemctl is-active gitlab-runner)"

log "Done. Verify on GitLab → Settings → CI/CD → Runners — the runner should show as online."
echo "Then push to main and click 'Run' on the deploy:vm job in the pipeline view."
