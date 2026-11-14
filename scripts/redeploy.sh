#!/usr/bin/env bash
set -euo pipefail

# Redeploy WMS: verify, commit, rebuild images, recreate app containers and
# confirm health. Pushes to origin only after the deploy is healthy.
#
# Usage:
#   scripts/redeploy.sh -m "feat: message" [-y] [--no-push] [--skip-tests]
#
# Options:
#   -m, --message MSG   Commit message (default: chore(deploy): <timestamp>)
#       --no-push       Do not push to origin after a healthy deploy
#       --skip-tests    Skip backend/frontend checks (faster, not recommended)
#   -y, --yes           Do not prompt before committing
#   -h, --help          Show this help
#
# Environment overrides:
#   BRANCH, SERVICES, HEALTH_TIMEOUT, GO_BIN, NODE_BIN_DIR

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

GO_BIN="${GO_BIN:-/home/ardi/.local/go/bin/go}"
NODE_BIN_DIR="${NODE_BIN_DIR:-/home/ardi/.local/node/bin}"
export PATH="$NODE_BIN_DIR:$PATH"

BRANCH="${BRANCH:-$(git branch --show-current 2>/dev/null || echo master)}"
SERVICES="${SERVICES:-backend worker frontend}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-180}"
COMMIT_MSG=""
DO_PUSH=1
RUN_TESTS=1
ASSUME_YES=0

log() { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }
die() { printf '\n\033[1;31mERROR: %s\033[0m\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--message) COMMIT_MSG="${2:-}"; shift 2;;
    --no-push) DO_PUSH=0; shift;;
    --skip-tests) RUN_TESTS=0; shift;;
    -y|--yes) ASSUME_YES=1; shift;;
    -h|--help) sed -n '4,20p' "$0"; exit 0;;
    *) die "unknown argument: $1 (use --help)";;
  esac
done

# --- preflight ---
[ -f .env ] || die ".env missing in $REPO_ROOT"
[ -f docker-compose.yml ] || die "docker-compose.yml missing"
command -v docker >/dev/null 2>&1 || die "docker not found"
docker compose version >/dev/null 2>&1 || die "docker compose plugin missing"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not a git repository"
command -v curl >/dev/null 2>&1 || die "curl not found"

set -a
# shellcheck disable=SC1091
. ./.env
set +a
API_HEALTH="http://127.0.0.1:${WMS_API_PORT:-8080}/healthz"
WEB_HEALTH="http://127.0.0.1:${WMS_FRONTEND_PORT:-3000}/healthz"

# --- verify ---
if [[ $RUN_TESTS -eq 1 ]]; then
  log "Backend build + tests"
  ( cd backend && "$GO_BIN" build ./... && "$GO_BIN" test ./... )
  log "Frontend lint + tests + build"
  ( cd frontend && npm run lint && npm test && npm run build )
else
  log "Skipping tests (--skip-tests)"
fi

# --- commit ---
if [[ -n "$(git status --porcelain)" ]]; then
  COMMIT_MSG="${COMMIT_MSG:-chore(deploy): $(date '+%Y-%m-%d %H:%M:%S')}"
  log "Staging changes"
  git add -A
  git status --short
  if [[ $ASSUME_YES -eq 0 ]]; then
    read -r -p "Commit as \"$COMMIT_MSG\"? [y/N] " ans
    [[ "$ans" =~ ^[Yy]$ ]] || die "aborted by user"
  fi
  git commit -m "$COMMIT_MSG"
else
  log "No source changes — redeploying current HEAD"
fi

COMMIT_SHA="$(git rev-parse --short HEAD)"

# --- deploy ---
log "Rebuilding + recreating: $SERVICES"
# shellcheck disable=SC2086
docker compose up -d --build $SERVICES

wait_health() {
  local name="$1" url="$2" deadline=$((SECONDS + HEALTH_TIMEOUT))
  log "Waiting for $name ($url)"
  until curl -fsS -o /dev/null "$url"; do
    if (( SECONDS >= deadline )); then
      docker compose ps
      die "$name did not become healthy within ${HEALTH_TIMEOUT}s"
    fi
    sleep 3
  done
  echo "  $name healthy"
}
wait_health backend "$API_HEALTH"
wait_health frontend "$WEB_HEALTH"

# --- push only after a healthy deploy ---
if [[ $DO_PUSH -eq 1 ]]; then
  log "Pushing $BRANCH -> origin"
  git push origin "$BRANCH"
else
  log "Skipping push (--no-push)"
fi

log "Done: commit=$COMMIT_SHA  api=$API_HEALTH  web=http://127.0.0.1:${WMS_FRONTEND_PORT:-3000}"
