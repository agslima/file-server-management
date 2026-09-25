# Supply Chain Security Runbook

## Purpose

Keep software supply-chain controls continuous by validating toolchain pins, SBOM generation, and SBOM signing readiness on every security posture review.

## Baseline command

```bash
./scripts/supply-chain-checks.sh
```

## Expected evidence

- `SUPPLY_CHAIN_CHECKS_OK` in stdout
- `SBOM_GENERATED` when `syft` is installed locally
- `SBOM_SIGNING_READY` when both `syft` and `cosign` are installed

## Signing drill story

1. Generate CycloneDX SBOM (`artifacts/sbom/repo.cdx.json`).
2. Sign the SBOM blob with `cosign sign-blob`.
3. Publish the SBOM + signature alongside release artifacts.
4. Verify signatures in release promotion checks.

When local tools are unavailable, retain the story output and execute full signing in CI/release infrastructure.
