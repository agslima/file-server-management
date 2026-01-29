# Storage Backends (Local / S3 / GCS)

This project supports multiple storage backends behind a single interface.

## Key point: ACL enforcement is identical across backends
Authorization is performed **before** issuing operations (API / interceptor / service layer).
The storage backend only executes the operation.

Backends:
- `local`: POSIX filesystem under `FILE_BASE_ROOT`
- `s3`: AWS S3 (or MinIO) bucket + optional prefix
- `gcs`: Google Cloud Storage bucket + optional prefix

## Folder semantics in object storage
Object stores use prefixes. We may create a zero-byte placeholder object ending with `/`.

## Atomic write semantics
Emulated by:
1) Upload temp key `<target>.tmp-<random>`
2) Copy temp → final
3) Delete temp (best effort)

## Environment variables
- `STORAGE_BACKEND=local|s3|gcs`

Local:
- `FILE_BASE_ROOT=/mnt/files`

S3:
- `S3_BUCKET`, `S3_REGION`, `S3_PREFIX` (optional), `S3_ENDPOINT` (optional for MinIO)
- AWS standard envs: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`

GCS:
- `GCS_BUCKET`, `GCS_PREFIX` (optional)
- Standard auth envs e.g. `GOOGLE_APPLICATION_CREDENTIALS`
