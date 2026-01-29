# Errors & Status Codes

## HTTP Status Codes
| HTTP Code | Meaning | Typical causes |
|---:|---|---|
| 400 | Bad Request | Invalid JSON, missing fields, invalid path |
| 401 | Unauthorized | Missing/invalid JWT, expired token |
| 403 | Forbidden | ACL/RBAC denies requested permission |
| 404 | Not Found | Task/upload/path not found (or not visible) |
| 409 | Conflict | Folder already exists |
| 413 | Payload Too Large | File exceeds limits |
| 415 | Unsupported Media Type | File type not allowed |
| 422 | Unprocessable Entity | Naming policy violated |
| 429 | Too Many Requests | Rate limiting (if enabled) |
| 500 | Internal Server Error | Unexpected server error |
| 503 | Service Unavailable | Redis/Postgres/storage unavailable |

## gRPC Code Mapping
| gRPC Code | HTTP | Meaning |
|---|---:|---|
| Unauthenticated | 401 | Missing/invalid auth |
| PermissionDenied | 403 | Access denied |
| InvalidArgument | 400/422 | Validation error |
| NotFound | 404 | Missing resource |
| AlreadyExists | 409 | Conflict |
| ResourceExhausted | 413/429 | Too large / rate limited |
| Internal | 500 | Server error |
| Unavailable | 503 | Dependency outage |
