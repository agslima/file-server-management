package auth

import (
	"context"
	"sort"
	"strings"
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
