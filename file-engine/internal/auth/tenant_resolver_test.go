package auth

import (
	"context"
	"testing"
)

func TestTenantResolverSourceOfTruth(t *testing.T) {
	r := NewInMemoryTenantResolver(map[string][]string{"alice": {"acme", "beta"}})
	tens, err := r.ResolveTenants(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tens) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(tens))
	}

	ok, err := r.UserHasTenant(context.Background(), "alice", "acme")
	if err != nil || !ok {
		t.Fatalf("expected alice to have acme tenant")
	}
	ok, err = r.UserHasTenant(context.Background(), "alice", "gamma")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("did not expect membership in gamma")
	}
}

func TestDenyAllTenantResolver(t *testing.T) {
	r := NewDenyAllTenantResolver()
	tens, err := r.ResolveTenants(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tens) != 0 {
		t.Fatalf("expected no tenants, got %d", len(tens))
	}

	ok, err := r.UserHasTenant(context.Background(), "alice", "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("deny-all resolver must deny tenant membership")
	}
}
