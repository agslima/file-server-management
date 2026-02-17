package di

import (
	"os"
	"testing"

	"github.com/example/file-engine/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBuildTenantResolverUsesPostgresWhenPoolConfigured(t *testing.T) {
	resolver := buildTenantResolver(&pgxpool.Pool{})
	if _, ok := resolver.(*auth.PostgresTenantResolver); !ok {
		t.Fatalf("expected postgres tenant resolver, got %T", resolver)
	}
}

func TestBuildTenantResolverFallsBackToDenyAllWithoutConfig(t *testing.T) {
	prev := os.Getenv("TENANT_MEMBERSHIPS")
	t.Cleanup(func() { _ = os.Setenv("TENANT_MEMBERSHIPS", prev) })
	_ = os.Unsetenv("TENANT_MEMBERSHIPS")

	resolver := buildTenantResolver(nil)
	if _, ok := resolver.(*auth.DenyAllTenantResolver); !ok {
		t.Fatalf("expected deny-all tenant resolver, got %T", resolver)
	}
}

func TestBuildTenantResolverFallsBackToInMemoryWhenEnvConfigured(t *testing.T) {
	prev := os.Getenv("TENANT_MEMBERSHIPS")
	t.Cleanup(func() { _ = os.Setenv("TENANT_MEMBERSHIPS", prev) })
	_ = os.Setenv("TENANT_MEMBERSHIPS", "alice=acme,beta")

	resolver := buildTenantResolver(nil)
	inmem, ok := resolver.(*auth.InMemoryTenantResolver)
	if !ok {
		t.Fatalf("expected in-memory tenant resolver, got %T", resolver)
	}

	okHas, err := inmem.UserHasTenant(t.Context(), "alice", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !okHas {
		t.Fatalf("expected membership to be loaded from TENANT_MEMBERSHIPS")
	}
}
