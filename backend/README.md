# Backend (Laravel API scaffold)

This backend now includes a minimal runnable API surface that proxies to File Engine:

- `POST /login`
- `POST /folders`
- `POST /uploads/initiate`
- `POST /uploads/complete`
- `GET /tasks/{id}`

## Local checks

```bash
cd backend
composer install
./scripts/smoke.sh
```

These tests validate controller behavior for the thin vertical slice contract.
