package auth

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresACLStore struct {
    pool *pgxpool.Pool
}

func NewPostgresACLStore(pool *pgxpool.Pool) *PostgresACLStore {
    return &PostgresACLStore{pool: pool}
}

func (s *PostgresACLStore) SetACL(acl ACL) error {
    ctx := context.Background()
    b, err := json.Marshal(acl.Permissions)
    if err != nil {
        return fmt.Errorf("marshal permissions: %w", err)
    }
    _, err = s.pool.Exec(ctx,
        `INSERT INTO acl_entries (path, principal_id, permissions) VALUES ($1,$2,$3::jsonb)`,
        acl.Path, acl.PrincipalID, string(b),
    )
    if err != nil {
        return fmt.Errorf("insert acl: %w", err)
    }
    return nil
}

func (s *PostgresACLStore) GetACLs(path string) []ACL {
    ctx := context.Background()
    rows, err := s.pool.Query(ctx,
        `SELECT path, principal_id, permissions FROM acl_entries WHERE path = $1`,
        path,
    )
    if err != nil {
        return nil
    }
    defer rows.Close()

    var out []ACL
    for rows.Next() {
        var p, principal string
        var permJSON []byte
        if err := rows.Scan(&p, &principal, &permJSON); err != nil {
            continue
        }
        perms := map[Permission]bool{}
        tmp := map[string]bool{}
        if err := json.Unmarshal(permJSON, &tmp); err == nil {
            for k, v := range tmp {
                perms[Permission(k)] = v
            }
        }
        out = append(out, ACL{Path: p, PrincipalID: principal, Permissions: perms})
    }
    return out
}

func (s *PostgresACLStore) GetACLsForPaths(paths []string) ([]ACL, error) {
    if len(paths) == 0 {
        return nil, nil
    }
    ctx := context.Background()
    rows, err := s.pool.Query(ctx,
        `SELECT path, principal_id, permissions FROM acl_entries WHERE path = ANY($1)`,
        paths,
    )
    if err != nil {
        return nil, fmt.Errorf("query acls: %w", err)
    }
    defer rows.Close()

    var out []ACL
    for rows.Next() {
        var p, principal string
        var permJSON []byte
        if err := rows.Scan(&p, &principal, &permJSON); err != nil {
            continue
        }
        perms := map[Permission]bool{}
        tmp := map[string]bool{}
        if err := json.Unmarshal(permJSON, &tmp); err == nil {
            for k, v := range tmp {
                perms[Permission(k)] = v
            }
        }
        out = append(out, ACL{Path: p, PrincipalID: principal, Permissions: perms})
    }
    return out, nil
}
