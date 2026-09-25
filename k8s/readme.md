# Kubernetes (kind) executable path

This directory provides an executable local Kubernetes path so deployment validation is not compose-only.

## Prerequisites

- `docker`
- `kind`
- `kubectl`

## One-command smoke (dev -> kind)

```bash
./scripts/k8s/kind_smoke.sh
```

The script:

1. creates/reuses a kind cluster,
2. builds and loads `file-engine:kind`,
3. applies `k8s/kind/file-server-kind.yaml`,
4. waits for rollouts,
5. validates `/healthz` and `/readyz` over port-forward.

Expected terminal marker: `KIND_SMOKE_OK`.

## Rollback drill

```bash
./scripts/drills/k8s_rollback_drill.sh
```

Expected terminal marker: `K8S_ROLLBACK_OK`.

## Secret bootstrap (required)

Create runtime secrets before or during apply:

```bash
kubectl create namespace file-platform --dry-run=client -o yaml | kubectl apply -f -
kubectl -n file-platform create secret generic file-engine-secrets \
  --from-literal=POSTGRES_DSN="postgres://fileengine:fileengine@postgres:5432/fileengine?sslmode=disable" \
  --from-literal=JWT_SECRET="dev-secret" \
  --dry-run=client -o yaml | kubectl apply -f -
```

`./scripts/k8s/kind_smoke.sh` also creates/updates this Secret from `POSTGRES_DSN` and `JWT_SECRET` env vars.
