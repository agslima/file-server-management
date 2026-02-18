-- Local/dev seed data for compose-based smoke/e2e checks.
-- Safe to rerun due ON CONFLICT guards.

INSERT INTO tenants (id, name)
VALUES ('acme', 'Acme')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, display_name)
VALUES ('dev-admin', 'dev-admin@example.com', 'Dev Admin')
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_tenants (user_id, tenant_id)
VALUES ('dev-admin', 'acme')
ON CONFLICT (user_id, tenant_id) DO NOTHING;
