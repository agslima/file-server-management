# Storage Backends (Local / S3 / GCS)

This project supports multiple storage backends behind the same `storage.Storage` interface.

## Key point: ACL enforcement is identical across backends

Authorization is performed **before** issuing operations (API / interceptor / service layer).
The storage backend simply executes the operation.

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
