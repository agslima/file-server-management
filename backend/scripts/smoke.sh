#!/usr/bin/env bash
set -euo pipefail

# Keep smoke deterministic in CI/local by default.
# Run backend<->engine integration only when explicitly requested.
RUN_BACKEND_INTEGRATION="${RUN_BACKEND_INTEGRATION:-0}"

# Avoid noisy xdebug connect warnings during smoke runs.
export XDEBUG_MODE="${XDEBUG_MODE:-off}"

if [[ ! -x "./vendor/bin/phpunit" ]]; then
  echo "[smoke] missing ./vendor/bin/phpunit. Run composer install --no-interaction before running ./scripts/smoke.sh." >&2
  exit 1
fi

echo "[smoke] running backend unit tests"
./vendor/bin/phpunit -c phpunit.xml tests/Unit

if [[ "$RUN_BACKEND_INTEGRATION" == "1" ]]; then
  echo "[smoke] running backend integration tests"
  ./vendor/bin/phpunit -c phpunit.xml tests/Integration
fi
