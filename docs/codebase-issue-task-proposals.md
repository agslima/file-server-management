# Codebase Issue Task Proposals

This note captures four focused follow-up tasks discovered during a quick repository review.

## 1) Typo task

### Task
Fix a wording typo/grammar issue in the integration test comment:
- Current comment: `enqueue -> worker process -> status success -> folder created.`
- Proposed: `enqueue -> worker processes task -> status success -> folder created.`

### Why
The current phrase reads like a noun instead of an action and is easy to misread while scanning the test.

### Evidence
- `file-engine/tests/integration/worker_integration_test.go` comment above `TestAsyncCreateFolderFlow`.

---

## 2) Bug task

### Task
Make `(*Worker).Start` exit promptly when `ctx` is cancelled instead of looping forever after `Pop` returns an error.

### Why
In the current implementation, cancellation can produce an error from `Pop`, but `Start` sleeps and continues indefinitely. That can hang shutdown paths and long-running tests.

### Evidence
- `file-engine/internal/worker/worker.go` retries forever on `Pop` errors with `time.Sleep(1 * time.Second)` and no `ctx.Done()` branch.

### Suggested acceptance criteria
- `Start` returns when `ctx.Done()` is closed.
- Add/adjust a unit test to prove cancellation stops the loop.

---

## 3) Comment/documentation discrepancy task

### Task
Align worker status documentation and behavior so they describe the same terminal status contract.

### Why
Repository docs repeatedly describe terminal statuses as `success` / `failed`, but the legacy worker implementation stores `error: <message>` strings. This makes status semantics inconsistent for anyone reading worker internals.

### Evidence
- Status contract in docs: `README.md` describes `queued → running → success | failed | quarantined`.
- Implementation mismatch: `file-engine/internal/worker/worker.go` sets `status = "error: " + err.Error()`.

### Suggested acceptance criteria
- Either normalize code to emit `failed` plus separate error message storage, or explicitly document this legacy status format where relevant.
- Include one regression test for the chosen contract.

---

## 4) Test improvement task

### Task
Improve the backend VS-001 E2E test by reducing dependence on shelling out to Docker from PHPUnit and by tightening failure diagnostics.

### Why
`VS001CreateFolderE2ETest` currently mixes HTTP validation with shell-based container probing via `shell_exec`, which can be brittle in CI and provides limited debugging context on failure.

### Evidence
- `backend/tests/Integration/VS001CreateFolderE2ETest.php` uses `shell_exec` for runtime checks and filesystem verification.

### Suggested acceptance criteria
- Keep the HTTP flow assertion in PHPUnit, but move environment/container checks behind a dedicated helper script or explicit test fixture abstraction.
- On failure, include last task payload/response details in assertions to speed triage.
