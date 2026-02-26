# Deployment profiles and env contract

This repository now defines explicit environment profiles so deployment assumptions are not compose-only:

- `dev`: local developer workflows and kind smoke
- `stage`: prod-like wiring with non-production credentials
- `prod`: strict profile requiring secret injection

## Files

- `env/.env.dev.example`
- `env/.env.stage.example`
- `env/.env.prod.example`

Use these files as templates only; do not commit real secret values.

## Config vs secrets contract

- **Config (committable):** endpoint hostnames, feature toggles, default ports, profile labels.
- **Secrets (non-committable):** passwords, API tokens, HMAC keys, bearer secrets.

Secrets must be injected via:

- local shell export (dev only),
- CI/CD secret manager,
- Kubernetes Secret objects.

## Required prod-like wiring

The script `scripts/check-runtime-wiring.sh` enforces required env in prod-like mode, including:

- OIDC/JWT validation wiring,
- OTEL exporter endpoint,
- immutable audit sink configuration,
- governance source envelope and HMAC key.

Run:

```bash
./scripts/check-runtime-wiring.sh --profile prod
```
