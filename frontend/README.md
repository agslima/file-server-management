# Frontend Demo Console (Thin Client)

This frontend is a lightweight static UI that demonstrates milestone 13 UX flows end-to-end without introducing a Node build system.

## Included flows

### 13.1 Minimal product UI
- OIDC login (`POST /api/login` via backend)
- Tenant selector
- Folder create + task status polling
- Upload flow (`initiate -> chunk -> complete`)
- Mutation actions (`move`, `delete`, `restore`)

### 13.2 Operator UI (admin-lite)
- Scan DLQ list + retry
- Quarantine cleanup trigger
- Effective policy view
- Drift status check
- Evidence pack pointer generation

## Run locally

From repo root:

```bash
python3 -m http.server 4173 --directory frontend
```

Then open `http://localhost:4173`.

> The UI calls backend (`http://localhost:8081/api`) and file-engine admin (`http://localhost:8080`) by default; update these fields in the Environment section as needed.

## Scope

This is intentionally a thin client and demo aid. Final authz/tenant enforcement remains in File Engine, not the browser.
