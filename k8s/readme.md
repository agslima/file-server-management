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
