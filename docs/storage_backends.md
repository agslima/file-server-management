# Storage Backends (Local / S3 / GCS)

This project supports multiple storage backends behind the same `storage.Storage` interface.

## Key point: ACL enforcement is identical across backends

Authorization is performed **before** issuing operations (API / interceptor / service layer).
The storage backend simply executes the operation.

## Backend parity matrix

| Capability | local | s3 | gcs | Notes |
| :-- | :--: | :--: | :--: | :-- |
| Path normalization (`\\`, duplicate slashes, clean root) | ✅ | ✅ | ✅ | Contract-enforced in adapter-level tests. |
| Deterministic `List` ordering by path | ✅ | ✅ | ✅ | Adapters normalize ordering so callers are backend-agnostic. |
| Read-after-write for atomic write path | ✅ | ✅ | ✅ | Validated in contract suite (`write` then immediate `exists/open`). |
| Atomic write semantics | ✅ | ✅ (temp+copy+delete) | ✅ (temp+copy+delete) | Object backends emulate atomicity. |
| Move semantics | ✅ | ✅ (copy+delete) | ✅ (copy+delete) | Source absent + destination present after move. |
| Metadata timestamps in list | ✅ | ✅ (`LastModified`, created fallback) | ✅ (`Updated`/`Created`) | Best-effort for cloud providers but required non-zero in contract tests. |
| Metadata checksum in list | ✅ (sha256, computed best-effort) | ✅ (ETag) | ✅ (MD5) | Format differs by backend; consumer treats as informational checksum. |
| Resumable/chunked upload semantics | ✅ (service-level) | ✅ (service-level) | ✅ (service-level) | Implemented in `UploadService` session/chunk/finalize API. |

## Consistency model and idempotency

Canonical model enforced by tests:

- **Read-after-write:** object must be visible immediately after successful `AtomicWrite`.
- **Move consistency:** after `Move`, destination is readable and source is absent.
- **List determinism:** `List` output is path-sorted regardless of backend.
- **Retry safety:** upload scanner retries are bounded and deterministic.
- **Idempotency:** upload idempotency key cannot be reused across different target paths; replay returns prior result.

## Metadata coverage (list results)

List responses include `size`, `is_dir`, and best-effort metadata fields (`modified_at`, `created_at`, `owner`, `group`, `checksum`). Coverage varies by backend:

| Backend | size | is_dir | modified_at | created_at | owner | group | checksum |
| :-- | :--: | :--: | :--: | :--: | :--: | :--: | :--: |
| `local` | ✅ | ✅ | ✅ | ✅ | ✅ (uid) | ✅ (gid) | ✅ (sha256 best-effort) |
| `s3` | ✅ | ✅ | ✅ (LastModified) | 🟡 (fallback to modified) | 🟡 (DisplayName if available) | ❌ | ✅ (ETag) |
| `gcs` | ✅ | ✅ | ✅ (Updated) | ✅ (Created) | ❌ | ❌ | ✅ (MD5) |

Baseline guarantees:

- Local backend metadata is validated in `TestLocalStorageListMetadata` and `TestListObjectsReturnsEntries`.
- Cross-backend canonical behavior is validated by storage contract suite tests.

Backends:
- `local`: POSIX filesystem under `FILE_BASE_ROOT`
- `s3`: AWS S3 (or MinIO) bucket + optional prefix
- `gcs`: Google Cloud Storage bucket + optional prefix

## CreateFolder semantics in object storage

Object stores are key-value stores. "Folders" are **prefixes**.
We optionally create a zero-byte placeholder object ending with `/`:
- S3 key: `<prefix>/path/to/folder/`
- GCS object: `<prefix>/path/to/folder/`

## AtomicWrite semantics in object storage

Atomic writes are emulated:
1) Upload to temp key: `<target>.tmp-<random>`
2) Copy temp -> final
3) Delete temp (best-effort)

## Environment variables

### Backend selection
- `STORAGE_BACKEND=local|s3|gcs`

### Local
- `FILE_BASE_ROOT=/mnt/files`

### S3
- `S3_BUCKET=...`
- `S3_REGION=...`
- `S3_PREFIX=` (optional)
- `S3_ENDPOINT=` (optional for MinIO)
- `AWS_ACCESS_KEY_ID=...` (optional if using instance roles)
- `AWS_SECRET_ACCESS_KEY=...`
- `AWS_SESSION_TOKEN=...`

### GCS
- `GCS_BUCKET=...`
- `GCS_PREFIX=` (optional)
- Standard GCP auth envs, e.g. `GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json`
