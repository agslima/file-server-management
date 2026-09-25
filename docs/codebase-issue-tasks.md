# Proposed Codebase Maintenance Tasks (Current Pass)

## 1) Typo task: normalize "JsonResponse" wording in controller PHPDoc

**Issue found**
- In `backend/app/Http/Controllers/ObjectMutationController.php`, one docblock reads `@return JsonResponse A JsonResponse containing ...`.
- This is grammatically awkward and inconsistent with nearby docs that use "A JSON response" phrasing.

**Proposed task**
- Update that annotation text to `@return JsonResponse A JSON response containing ...` (or equivalent concise wording), keeping behavior claims unchanged.

**Why this matters**
- Improves readability/consistency in IDE hover docs and generated API documentation.

## 2) Bug task: remove misleading fallback in `workflow_dispatch` modeling

**Issue found**
- In `tests/test_ci_pr_security_scan_change_detection.py`, `compute_outputs(..., event_name="workflow_dispatch")` falls back to `changed_files` when `repo_files` is omitted.
- Real workflow behavior for `workflow_dispatch` computes docker files by scanning the repository (`find . -type f ...`), not by using changed files.

**Proposed task**
- Require explicit `repo_files` for dispatch-mode computation (or simulate repository scan directly in the helper), and fail fast if dispatch mode is requested without repository context.

**Why this matters**
- Prevents false confidence from tests that pass under a helper-specific fallback not present in CI.

## 3) Comment/documentation discrepancy task: refresh stale issue-tracking doc

**Issue found**
- The previous `docs/codebase-issue-tasks.md` version listed tasks that are now already implemented (PHPDoc duplication removed, Dockerfile root match fix, workflow scope alignment, and dispatch coverage).

**Proposed task**
- Keep this document as a "current findings" list only:
  - remove completed items,
  - add discovery date/owner/status fields,
  - and archive completed findings in a separate historical section (or changelog entry).

**Why this matters**
- Avoids stale guidance and keeps maintenance tracking trustworthy.

## 4) Test-improvement task: add parity assertion against workflow-defined globs

**Issue found**
- The local helper duplicates workflow glob intent in Python constants, which can drift when workflow filters are changed.

**Proposed task**
- Add a parity test that verifies local constants mirror `.github/workflows/ci-pr-security-scan.yaml` (for `backend`, `frontend`, `file_engine`, and docker globs), or generate the local constants from a single canonical fixture.

**Why this matters**
- Catches workflow/test drift automatically during CI and reduces maintenance overhead.
