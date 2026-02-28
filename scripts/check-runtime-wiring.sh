#!/usr/bin/env bash
set -euo pipefail

PROFILE="dev"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      if [[ $# -lt 2 || -z "${2:-}" || "${2:-}" == -* ]]; then
        echo "missing value for --profile (usage: --profile <name>)" >&2
        exit 2
      fi
      PROFILE="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

require_non_empty() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "missing required env: ${name}" >&2
    return 1
  fi
}

require_url() {
  local name="$1"
  require_non_empty "$name" || return 1
  if [[ ! "${!name}" =~ ^https?:// ]]; then
    echo "env must be http(s) URL: ${name}" >&2
    return 1
  fi
}

fail=0
for var in JWT_JWKS_URL JWT_ISSUER JWT_AUDIENCE OTEL_EXPORTER_OTLP_ENDPOINT GOVERNANCE_POLICY_SOURCE GOVERNANCE_POLICY_SOURCE_HMAC_KEY AUDIT_IMMUTABLE_SINK_TYPE; do
  require_non_empty "$var" || fail=1
done

require_url JWT_JWKS_URL || fail=1
require_url JWT_ISSUER || fail=1
require_url OTEL_EXPORTER_OTLP_ENDPOINT || fail=1

case "${AUDIT_IMMUTABLE_SINK_TYPE:-}" in
  siem)
    require_url AUDIT_SIEM_ENDPOINT || fail=1
    require_non_empty AUDIT_SIEM_API_KEY || fail=1
    ;;
  loki)
    require_url AUDIT_LOKI_ENDPOINT || fail=1
    ;;
  s3_worm)
    require_non_empty AUDIT_S3_BUCKET || fail=1
    ;;
  file|bucket)
    require_non_empty AUDIT_IMMUTABLE_SINK_PATH || fail=1
    ;;
  *)
    echo "unsupported AUDIT_IMMUTABLE_SINK_TYPE=${AUDIT_IMMUTABLE_SINK_TYPE:-}" >&2
    fail=1
    ;;
esac

if [[ ! -f "${GOVERNANCE_POLICY_SOURCE:-}" ]]; then
  echo "governance source envelope file not found: ${GOVERNANCE_POLICY_SOURCE:-}" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  echo "RUNTIME_WIRING_FAIL profile=${PROFILE}" >&2
  exit 1
fi

echo "RUNTIME_WIRING_OK profile=${PROFILE}"
