# Proposed maintenance tasks

## 1) Typo fix task
**Title:** Fix typo in backend route documentation for upload completion endpoint.

**Issue:** `backend/README.md` currently documents `POST /uploads/complete`, but the runnable route pattern includes an upload identifier segment (`/uploads/{id}/complete`). This looks like a path typo in the docs list and can mislead API consumers using the thin backend slice.

**Where observed:**
- `backend/README.md` route list.
- `backend/api.php` route matcher for upload completion.

**Suggested outcome:** Update the backend README route list entry to `POST /uploads/{id}/complete` and keep path examples aligned with actual request routing.

---

## 2) Bug fix task
**Title:** Fix incorrect constructor argument order when bootstrapping `FileEngineService` in `backend/api.php`.

**Issue:** `FileEngineService` expects constructor parameters in this order: `(HttpFactory $http, ?string $baseUrl, ?string $adminBaseUrl, ?string $bearerToken)`. In `backend/api.php`, the third argument currently passes `FILE_ENGINE_BEARER_TOKEN`, so the token is being assigned to `adminBaseUrl` instead of `bearerToken`.

**Risk:** Admin requests may be sent to an invalid base URL (the bearer token string), and configured auth may not be applied as intended.

**Suggested outcome:** Use named args or pass all four arguments in the correct order so `bearerToken` is set properly and `adminBaseUrl` remains optional/empty unless explicitly configured.

---

## 3) Comment/documentation discrepancy task
**Title:** Align `FileEngineService` header-forwarding docs with actual Authorization behavior.

**Issue:** Multiple PHPDoc blocks say `Authorization` is an optional forwarded trace header, but `filteredTraceHeaders()` only forwards `Authorization` when no service-level bearer token is configured.

**Where observed:**
- PHPDoc for methods like `createFolder`, `completeUpload`, `deleteObject`, and `getTask` mention forwarding `Authorization`.
- `filteredTraceHeaders()` conditionally suppresses request `Authorization` if `$this->bearerToken` is set.

**Suggested outcome:** Clarify docs/comments to explicitly state precedence: configured service bearer token overrides caller `Authorization`; caller `Authorization` is only forwarded when service token is empty.

---

## 4) Test improvement task
**Title:** Reduce polling flakiness by replacing fixed sleeps with eventual assertions in async tests.

**Issue:** Several tests use fixed `time.Sleep(...)` polling loops for async progression. This can cause flakes (too short on slow CI) or unnecessary test latency (too long locally).

**Where observed:**
- `file-engine/tests/integration/async_mutation_expansion_integration_test.go`
- `file-engine/tests/integration/worker_integration_test.go`
- `file-engine/internal/server/admin_http_test.go`
- `file-engine/internal/services/upload_service_test.go`

**Suggested outcome:** Replace ad-hoc sleep loops with bounded eventual assertions (e.g., helper retry loop with timeout + interval), and include useful timeout diagnostics to improve CI debuggability.
