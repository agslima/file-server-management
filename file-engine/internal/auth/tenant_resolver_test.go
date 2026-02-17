package auth

import (
	"context"
	"strings"
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

func TestPostgresTenantResolverNilPool(t *testing.T) {
	r := NewPostgresTenantResolver(nil)
	_, err := r.ResolveTenants(context.Background(), "alice")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured error, got %v", err)
	}

	ok, err := r.UserHasTenant(context.Background(), "alice", "acme")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured error, got %v", err)
	}
	if ok {
		t.Fatalf("expected false membership when resolver is not configured")
	}
}
