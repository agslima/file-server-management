# API Versioning & Compatibility Policy

## Purpose

This policy defines compatibility guarantees for external consumers of File Engine HTTP/gRPC APIs.

## HTTP endpoint versioning (`/v1`)

- Stable public HTTP routes are namespaced under `/v1`.
- **No breaking change is allowed within `/v1`**.
- Breaking changes include:
  - removing or renaming a route,
  - removing response fields,
  - changing field type/shape incompatibly,
  - changing stable error envelope fields (`error.code`, `error.reason`, `error.request_id`, `error.correlation_id`) for authz-deny paths.
- Any breaking behavior requires a new API version namespace (for example `/v2`).

## Proto/gRPC evolution rules

- Proto changes must preserve backward compatibility for existing field numbers and service method contracts.
- Never reuse or repurpose field numbers; reserve removed field numbers/names.
- Additive changes (new optional fields, new RPCs) are allowed in the current major version.
- Breaking RPC or message changes require a new versioned service/package and corresponding HTTP namespace transition plan.
- Proto mirrors must stay synchronized:
  - `file-engine/api/proto/fileengine.proto`
  - `file-engine/proto/fileengine.proto`

## Compatibility gate

Compatibility is enforced via frozen golden fixtures for key endpoints:
- upload lifecycle (`initiate` and `complete` response contracts),
- readiness endpoint (`/readyz`),
- authz deny envelope shape (`AUTHZ_DENY`).

Run locally:

```bash
./scripts/check-api-compatibility.sh
```

## Deprecation process

1. Mark behavior/field as deprecated in docs and changelog.
2. Add migration guidance and replacement path.
3. Keep deprecated behavior for at least one minor release train.
4. Remove only when a new major API version is available and communicated.
