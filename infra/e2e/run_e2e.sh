#!/usr/bin/env bash
# Brings up the full compose stack, waits for readiness, runs the end-to-end
# test, and tears everything down. Run from the repo root:
#
#   JWT_SECRET=$(openssl rand -hex 32) infra/e2e/run_e2e.sh
set -euo pipefail

COMPOSE="docker compose -f infra/docker-compose.yml"
export COMPOSE
export JWT_SECRET="${JWT_SECRET:-e2e-only-change-me-0123456789abcdef0123456789}"

cleanup() {
  echo "[e2e] tearing down stack"
  $COMPOSE down -v || true
}
trap cleanup EXIT

echo "[e2e] building + starting stack"
$COMPOSE up --build -d

wait_for() {
  local name="$1" url="$2" tries=60
  echo "[e2e] waiting for $name ($url)"
  until curl -fsS "$url" >/dev/null 2>&1; do
    tries=$((tries - 1))
    if [ "$tries" -le 0 ]; then
      echo "[e2e] $name did not become ready"; $COMPOSE logs "$name" | tail -50; exit 1
    fi
    sleep 3
  done
}

wait_for api-service "http://localhost:8080/actuator/health/readiness"
wait_for gateway "http://localhost:8090/healthz"

# The test uses only the Python standard library — no pip install needed.
echo "[e2e] running e2e test"
python3 infra/e2e/e2e_test.py
