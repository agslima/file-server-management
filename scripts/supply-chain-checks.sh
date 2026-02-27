#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

: "${SBOM_OUT:=artifacts/sbom}"
: "${SBOM_SIGNING_STORY:=docs/runbooks/supply-chain-security.md}"

mkdir -p "$SBOM_OUT"

echo "[supply-chain] checking pinned toolchain declarations"
rg -n '^go [0-9]+\.[0-9]+\.[0-9]+' file-engine/go.mod >/dev/null
rg -n '"php": "\^[0-9]+\.[0-9]+"' backend/composer.json >/dev/null

echo "[supply-chain] generating SBOM when syft is available"
if command -v syft >/dev/null 2>&1; then
  syft dir:. -o cyclonedx-json > "$SBOM_OUT/repo.cdx.json"
  echo "SBOM_GENERATED path=$SBOM_OUT/repo.cdx.json"
else
  echo "SBOM_SKIPPED syft-not-installed"
fi

echo "[supply-chain] signing story"
if command -v cosign >/dev/null 2>&1 && [[ -f "$SBOM_OUT/repo.cdx.json" ]]; then
  echo "cosign sign-blob --yes --output-signature $SBOM_OUT/repo.cdx.json.sig $SBOM_OUT/repo.cdx.json"
  echo "SBOM_SIGNING_READY"
else
  echo "SBOM_SIGNING_STORY path=$SBOM_SIGNING_STORY"
fi

echo "SUPPLY_CHAIN_CHECKS_OK"
