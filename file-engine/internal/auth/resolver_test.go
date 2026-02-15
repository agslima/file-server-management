package auth

import "testing"

func TestRBACFallback(t *testing.T) {
	store := NewInMemoryACLStore()
	ctx := AuthContext{UserID: "42", Roles: []string{"viewer"}}

	if !CanAccess(ctx, "/any/path", PermRead, store) {
		t.Fatal("viewer should have read access via RBAC")
	}

	if CanAccess(ctx, "/any/path", PermWrite, store) {
		t.Fatal("viewer should not have write access")
	}
}

func TestUserACLOverridesRBAC(t *testing.T) {
	store := NewInMemoryACLStore()
	_ = store.SetACL(ACL{Path: "/projects/alpha", PrincipalID: "user:42", Permissions: map[Permission]bool{PermWrite: true}})

	ctx := AuthContext{UserID: "42", Roles: []string{"viewer"}}
	if !CanAccess(ctx, "/projects/alpha", PermWrite, store) {
		t.Fatal("explicit user ACL should override RBAC")
	}
}

func TestACLPathInheritance(t *testing.T) {
	store := NewInMemoryACLStore()
	_ = store.SetACL(ACL{Path: "/projects", PrincipalID: "role:editor", Permissions: map[Permission]bool{PermWrite: true}})

	ctx := AuthContext{UserID: "99", Roles: []string{"editor"}}
	if !CanAccess(ctx, "/projects/alpha/docs", PermWrite, store) {
		t.Fatal("ACL should be inherited from parent path")
	}
}

func TestUserDenyPrecedesRoleAllowAndRBAC(t *testing.T) {
	store := NewInMemoryACLStore()
	_ = store.SetACL(ACL{Path: "/tenants/acme", PrincipalID: "user:alice", Permissions: map[Permission]bool{PermWrite: false}})
	_ = store.SetACL(ACL{Path: "/tenants/acme", PrincipalID: "role:admin", Permissions: map[Permission]bool{PermWrite: true}})

	ctx := AuthContext{UserID: "alice", Roles: []string{"admin"}}
	if CanAccess(ctx, "/tenants/acme/docs", PermWrite, store) {
		t.Fatal("user deny must win over role allow and RBAC")
	}
}

func TestRoleDenyPrecedesRoleAllowAtSameLevel(t *testing.T) {
	store := NewInMemoryACLStore()
	_ = store.SetACL(ACL{Path: "/tenants/acme", PrincipalID: "role:editor", Permissions: map[Permission]bool{PermWrite: false}})
	_ = store.SetACL(ACL{Path: "/tenants/acme", PrincipalID: "role:admin", Permissions: map[Permission]bool{PermWrite: true}})

	ctx := AuthContext{UserID: "bob", Roles: []string{"editor", "admin"}}
	if CanAccess(ctx, "/tenants/acme/docs", PermWrite, store) {
		t.Fatal("role deny should win over role allow at same path level")
	}
}

func TestDenyByDefault(t *testing.T) {
	store := NewInMemoryACLStore()
	ctx := AuthContext{UserID: "13", Roles: nil}
	if CanAccess(ctx, "/secret", PermRead, store) {
		t.Fatal("access should be denied by default")
	}
}
