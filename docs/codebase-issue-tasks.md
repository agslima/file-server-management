# Proposed Codebase Maintenance Tasks

## 1) Typo task: remove duplicated type name in a PHPDoc `@return`

**Issue found**
- `ObjectMutationController::restore` has `@return JsonResponse JsonResponse ...`, where `JsonResponse` is duplicated in the same sentence.

**Proposed task**
- Update the docblock to use a single `JsonResponse` token and keep the rest of the return description intact.

**Why this matters**
- Reduces documentation noise and avoids confusion in generated API docs / IDE hover text.

## 2) Bug task: fix Dockerfile glob matching in local change-detection helper

**Issue found**
- The helper in `tests/test_ci_pr_security_scan_change_detection.py` uses `PurePosixPath.match("**/Dockerfile*")` for Dockerfile detection.
- In Python path matching semantics, `**/Dockerfile*` does not match a root-level `Dockerfile` path, which can produce false negatives versus the intent of the workflow glob.

**Proposed task**
- Replace glob matching logic with a matcher that treats root-level and nested Dockerfiles consistently (for example, explicit checks for basename + suffix, or normalize patterns before matching).

**Why this matters**
- Prevents under-detection of Docker-related changes when files are located at repository root.

## 3) Comment/documentation discrepancy task: align test assumptions with workflow scope

**Issue found**
- The test file states: `Keep these patterns in sync with .github/workflows/ci-pr-security-scan.yaml`.
- However, test `FILTERS` only track dependency manifests (`composer.lock`, `package-lock.json`, `go.mod`, etc.), while the workflow currently flags any changes under `backend/**`, `frontend/**`, and `file-engine/**`.

**Proposed task**
- Update `FILTERS` and test cases to mirror the workflow behavior exactly, or narrow the workflow patterns and document the rationale. Keep one canonical source of truth.

**Why this matters**
- Avoids a false sense of coverage: tests can pass while workflow behavior has drifted.

## 4) Test-improvement task: add workflow-dispatch behavior coverage

**Issue found**
- The workflow has a special branch that force-enables backend/frontend/file_engine/docker on `workflow_dispatch`.
- The local verification script currently models only changed-file based behavior and does not exercise this dispatch mode.

**Proposed task**
- Extend the local change-detection test helper to accept event type (e.g., `pull_request` vs `workflow_dispatch`) and add explicit assertions for dispatch-mode outputs.

**Why this matters**
- Ensures the local test continues to validate both PR-triggered and manual-triggered workflow semantics.
