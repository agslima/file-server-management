# Authentication & Authorization

## Authentication (JWT Bearer)
All endpoints require:
```
Authorization: Bearer <JWT>
```

### Required claims
- `sub` → user identifier (`AuthContext.UserID`)
- `roles` → array of role names (`AuthContext.Roles`)

### Optional (recommended)
- `iss` (issuer) and `aud` (audience), validated if configured.

### Example JWT payload
```json
{
  "sub": "user-42",
  "roles": ["admin","editor"],
  "iss": "your-issuer",
  "aud": "file-engine",
  "exp": 1896144000
}
```

## Authorization (RBAC + Path-based ACL)
Authorization is enforced before operations are executed/enqueued.

Resolution order:
1. Closest ACL for `user:<sub>` on path
2. Closest ACL for `role:<role>` on path
3. RBAC fallback (role defaults)
4. Deny by default

Inheritance walks up the path:
`/a/b/c` → `/a/b` → `/a` → `/`
