#!/usr/bin/env bash
set -euo pipefail

COMPOSE="docker compose --profile oidc"
E2E_TIMEOUT_SECONDS="${OIDC_E2E_TIMEOUT_SECONDS:-180}"

trap '${COMPOSE} down -v --remove-orphans' EXIT

${COMPOSE} build keycloak postgres redis file-engine file-engine-worker
${COMPOSE} up -d keycloak postgres redis file-engine file-engine-worker

./scripts/wait-for-http.sh http://localhost:8082/realms/file-engine/.well-known/openid-configuration 120
./scripts/wait-for-http.sh http://localhost:8080/readyz 120

timeout --signal=TERM "${E2E_TIMEOUT_SECONDS}" ./scripts/e2e/oidc_login_and_call_engine.sh
