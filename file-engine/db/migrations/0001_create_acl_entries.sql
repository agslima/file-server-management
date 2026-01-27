-- ACL persistence for File Engine

CREATE TABLE IF NOT EXISTS acl_entries (
  id BIGSERIAL PRIMARY KEY,
  path TEXT NOT NULL,
  principal_id TEXT NOT NULL,
  permissions JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_acl_entries_path ON acl_entries(path);
CREATE INDEX IF NOT EXISTS idx_acl_entries_path_principal ON acl_entries(path, principal_id);
