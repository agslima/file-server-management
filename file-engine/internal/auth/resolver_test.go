package auth

import "testing"

func TestRBACFallback(t *testing.T) {
    store := NewInMemoryACLStore()
    ctx := AuthContext{
        UserID: "42",
        Roles:  []string{"viewer"},
    }

    if !CanAccess(ctx, "/any/path", PermRead, store) {
        t.Fatal("viewer should have read access via RBAC")
    }

    if CanAccess(ctx, "/any/path", PermWrite, store) {
        t.Fatal("viewer should not have write access")
    }
}

func TestUserACLOverridesRBAC(t *testing.T) {
    store := NewInMemoryACLStore()
    store.SetACL(ACL{
        Path:        "/projects/alpha",
        PrincipalID: "user:42",
        Permissions: map[Permission]bool{
            PermWrite: true,
        },
    })

    ctx := AuthContext{
        UserID: "42",
        Roles:  []string{"viewer"},
    }

    if !CanAccess(ctx, "/projects/alpha", PermWrite, store) {
        t.Fatal("explicit user ACL should override RBAC")
    }
}

func TestPathInheritance(t *testing.T) {
    store := NewInMemoryACLStore()
    store.SetACL(ACL{
        Path:        "/projects",
        PrincipalID: "role:editor",
        Permissions: map[Permission]bool{
            PermWrite: true,
        },
    })

    ctx := AuthContext{
        UserID: "99",
        Roles:  []string{"editor"},
    }

    if !CanAccess(ctx, "/projects/alpha/docs", PermWrite, store) {
        t.Fatal("ACL should be inherited from parent path")
    }
}

func TestDenyByDefault(t *testing.T) {
    store := NewInMemoryACLStore()
    ctx := AuthContext{
        UserID: "13",
        Roles:  []string{},
    }

    if CanAccess(ctx, "/secret", PermRead, store) {
        t.Fatal("access should be denied by default")
    }
}
