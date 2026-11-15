#!/usr/bin/env bash
set -euo pipefail

# Deploy WMS from origin/master: pull, rebuild images, recreate app containers,
# then wait for backend and frontend health. Used by the self-hosted GitHub
# Actions runner on push. Manual use: scripts/deploy.sh
#
# This script never commits or pushes. It mirrors the server checkout to the
# remote branch and preserves untracked files such as .env.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRANCH="${DEPLOY_BRANCH:-master}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-180}"

log() { printf '\n==> %s\n' "$*"; }

cd "$REPO_ROOT"

log "Sync origin/$BRANCH"
git fetch --prune origin "$BRANCH"
git reset --hard "origin/$BRANCH"

[ -f .env ] || { echo ".env missing in $REPO_ROOT" >&2; exit 1; }
set -a
# shellcheck disable=SC1091
. ./.env
set +a

log "Rebuild + recreate app services"
docker compose up -d --build backend worker frontend

API_HEALTH="http://127.0.0.1:${WMS_API_PORT:-8080}/healthz"
WEB_HOST="${WMS_BIND_IP:-127.0.0.1}"
if [ "$WEB_HOST" = "0.0.0.0" ]; then
  WEB_HOST="127.0.0.1"
fi
WEB_HEALTH="http://${WEB_HOST}:${WMS_FRONTEND_PORT:-3000}/healthz"

wait_health() {
  local service="$1" url="$2" deadline=$((SECONDS + HEALTH_TIMEOUT))
  log "Waiting for $service ($url)"
  until curl -fsS -o /dev/null "$url"; do
    if (( SECONDS >= deadline )); then
      docker compose ps
      docker compose logs --tail=100 "$service" || true
      echo "$service not healthy within ${HEALTH_TIMEOUT}s" >&2
      exit 1
    fi
    sleep 3
  done
  echo "  $service healthy"
}

wait_health backend "$API_HEALTH"
wait_health frontend "$WEB_HEALTH"

log "Deploy complete: $(git rev-parse --short HEAD)"
