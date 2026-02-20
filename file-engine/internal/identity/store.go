// Package identity provides persistence for tenant, user, and role identity data.
package identity

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }
func (s *Store) Ready() bool             { return s != nil && s.pool != nil }

func (s *Store) CreateTenant(ctx context.Context, id, name string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO tenants (id, name) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`, id, name)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, id, email, displayName string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1,$2,$3)
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name`, id, email, displayName)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Store) CreateMembership(ctx context.Context, userID, tenantID string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO user_tenants (user_id, tenant_id) VALUES ($1,$2) ON CONFLICT (user_id, tenant_id) DO NOTHING`, userID, tenantID)
	if err != nil {
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

func (s *Store) RevokeMembership(ctx context.Context, userID, tenantID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_tenants WHERE user_id=$1 AND tenant_id=$2`, userID, tenantID)
	if err != nil {
		return fmt.Errorf("revoke membership: %w", err)
	}
	return nil
}

func (s *Store) UpsertRole(ctx context.Context, roleID, description string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO roles (id, description) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description`, roleID, description)
	if err != nil {
		return fmt.Errorf("upsert role: %w", err)
	}
	return nil
}

func (s *Store) AssignRole(ctx context.Context, userID, tenantID, roleID string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO user_tenant_roles (user_id, tenant_id, role_id) VALUES ($1,$2,$3) ON CONFLICT (user_id, tenant_id, role_id) DO NOTHING`, userID, tenantID, roleID)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

type AccessReviewRow struct {
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id"`
	Email      string     `json:"email,omitempty"`
	RoleID     string     `json:"role_id,omitempty"`
	LastAccess *time.Time `json:"last_access,omitempty"`
}

func (s *Store) AccessReview(ctx context.Context, tenantID string) ([]AccessReviewRow, error) {
	rows, err := s.pool.Query(ctx, `
SELECT ut.tenant_id, ut.user_id, COALESCE(u.email, ''), COALESCE(utr.role_id, ''),
(
	SELECT MAX(ae.created_at) FROM audit_events ae
	WHERE ae.metadata->>'tenant_id' = ut.tenant_id AND ae.metadata->>'actor_id' = ut.user_id
) AS last_access
FROM user_tenants ut
LEFT JOIN users u ON u.id = ut.user_id
LEFT JOIN user_tenant_roles utr ON utr.user_id = ut.user_id AND utr.tenant_id = ut.tenant_id
WHERE ($1 = '' OR ut.tenant_id = $1)
ORDER BY ut.tenant_id ASC, ut.user_id ASC, utr.role_id ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query access review: %w", err)
	}
	defer rows.Close()
	var out []AccessReviewRow
	for rows.Next() {
		var r AccessReviewRow
		if err := rows.Scan(&r.TenantID, &r.UserID, &r.Email, &r.RoleID, &r.LastAccess); err != nil {
			return nil, fmt.Errorf("scan access review: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
