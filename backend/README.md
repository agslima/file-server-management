# Backend (Laravel API scaffold)

This backend now includes a minimal runnable API surface that proxies to File Engine:

- `POST /login`
- `POST /folders`
- `POST /uploads/initiate`
- `POST /uploads/{id}/complete`
- `GET /tasks/{id}`

## Local checks

```bash
cd backend
composer install
./scripts/smoke.sh
```

Host PHP must be `>=8.2` (Laravel 12 requirement). If your local PHP is older, run checks in Docker:

```bash
docker compose run --rm --no-deps backend sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'
```

These tests validate controller behavior for the thin vertical slice contract.
