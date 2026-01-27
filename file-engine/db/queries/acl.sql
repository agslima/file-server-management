-- Insert
INSERT INTO acl_entries (path, principal_id, permissions)
VALUES ($1, $2, $3::jsonb);

-- Get by exact path
SELECT path, principal_id, permissions
FROM acl_entries
WHERE path = $1;

-- Get for multiple paths (inheritance)
SELECT path, principal_id, permissions
FROM acl_entries
WHERE path = ANY($1);
