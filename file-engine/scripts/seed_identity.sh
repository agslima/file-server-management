#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_DSN:=postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable}"

psql "$POSTGRES_DSN" <<'SQL'
INSERT INTO tenants (id, name) VALUES
  ('acme', 'Acme Corp'),
  ('beta', 'Beta Org')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, display_name) VALUES
  ('dev-admin', 'dev-admin@example.com', 'Dev Admin'),
  ('dev-analyst', 'dev-analyst@example.com', 'Dev Analyst')
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_tenants (user_id, tenant_id) VALUES
  ('dev-admin', 'acme'),
  ('dev-analyst', 'beta')
ON CONFLICT (user_id, tenant_id) DO NOTHING;

INSERT INTO roles (id, description) VALUES
  ('admin', 'Tenant administrator'),
  ('viewer', 'Read-only')
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO user_tenant_roles (user_id, tenant_id, role_id) VALUES
  ('dev-admin', 'acme', 'admin'),
  ('dev-analyst', 'beta', 'viewer')
ON CONFLICT (user_id, tenant_id, role_id) DO NOTHING;
SQL

echo "Identity seed complete"
