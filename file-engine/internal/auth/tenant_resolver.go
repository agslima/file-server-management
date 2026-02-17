package auth

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantResolver is the server-side source-of-truth for tenant membership.
// It must not trust tenant scope supplied by client or JWT claims.
type TenantResolver interface {
	ResolveTenants(ctx context.Context, userID string) ([]string, error)
	UserHasTenant(ctx context.Context, userID, tenantID string) (bool, error)
}

type InMemoryTenantResolver struct {
	memberships map[string]map[string]struct{}
}

func NewInMemoryTenantResolver(seed map[string][]string) *InMemoryTenantResolver {
	m := map[string]map[string]struct{}{}
	for user, tenants := range seed {
		if _, ok := m[user]; !ok {
			m[user] = map[string]struct{}{}
		}
		for _, t := range tenants {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			m[user][t] = struct{}{}
		}
	}
	return &InMemoryTenantResolver{memberships: m}
}

func (r *InMemoryTenantResolver) ResolveTenants(_ context.Context, userID string) ([]string, error) {
	set := r.memberships[userID]
	if len(set) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}

func (r *InMemoryTenantResolver) UserHasTenant(_ context.Context, userID, tenantID string) (bool, error) {
	set := r.memberships[userID]
	_, ok := set[tenantID]
	return ok, nil
}

type PostgresTenantResolver struct {
	pool *pgxpool.Pool
}

func NewPostgresTenantResolver(pool *pgxpool.Pool) *PostgresTenantResolver {
	return &PostgresTenantResolver{pool: pool}
}

func (r *PostgresTenantResolver) ResolveTenants(ctx context.Context, userID string) ([]string, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("postgres tenant resolver not configured")
	}
	rows, err := r.pool.Query(ctx,
		`SELECT tenant_id FROM user_tenants WHERE user_id = $1 ORDER BY tenant_id ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query user_tenants: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			return nil, fmt.Errorf("scan tenant_id: %w", err)
		}
		out = append(out, tenantID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenants: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (r *PostgresTenantResolver) UserHasTenant(ctx context.Context, userID, tenantID string) (bool, error) {
	if r.pool == nil {
		return false, fmt.Errorf("postgres tenant resolver not configured")
	}
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_tenants WHERE user_id = $1 AND tenant_id = $2)`,
		userID,
		tenantID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query membership: %w", err)
	}
	return exists, nil
}

// DenyAllTenantResolver is the secure default when source-of-truth mappings are unavailable.
type DenyAllTenantResolver struct{}

func NewDenyAllTenantResolver() *DenyAllTenantResolver { return &DenyAllTenantResolver{} }

func (r *DenyAllTenantResolver) ResolveTenants(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (r *DenyAllTenantResolver) UserHasTenant(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// AllowAllTenantResolver is a compatibility fallback for environments
// that have not configured source-of-truth tenant mappings yet.
type AllowAllTenantResolver struct{}

func NewAllowAllTenantResolver() *AllowAllTenantResolver { return &AllowAllTenantResolver{} }

func (r *AllowAllTenantResolver) ResolveTenants(_ context.Context, _ string) ([]string, error) {
	return []string{"*"}, nil
}

func (r *AllowAllTenantResolver) UserHasTenant(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
