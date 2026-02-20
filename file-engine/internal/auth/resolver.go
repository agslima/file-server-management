package auth

import "path/filepath"

type AuthContext struct {
	UserID  string
	ActorID string
	Email   string
	Groups  []string
	Roles   []string
}

func (a AuthContext) EffectiveActorID() string {
	if a.ActorID != "" {
		return a.ActorID
	}
	return a.UserID
}

func parentPaths(path string) []string {
	var paths []string
	for {
		paths = append(paths, path)
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	return paths
}

type ACLBatchStore interface {
	ACLStore
	GetACLsForPaths(paths []string) ([]ACL, error)
}

// CanAccess applies explicit precedence:
// 1) User ACL on closest matching path (allow/deny)
// 2) Role ACL on closest matching path (deny wins over allow)
// 3) RBAC fallback
// 4) Deny by default
func CanAccess(ctx AuthContext, path string, perm Permission, store ACLStore) bool {
	paths := parentPaths(path)

	if bs, ok := store.(ACLBatchStore); ok {
		acls, err := bs.GetACLsForPaths(paths)
		if err == nil {
			byPath := map[string][]ACL{}
			for _, a := range acls {
				byPath[a.Path] = append(byPath[a.Path], a)
			}
			if decision, decided := decideByACL(ctx, paths, perm, byPath); decided {
				return decision
			}
			return decideByRBAC(ctx, perm)
		}
	}

	byPath := map[string][]ACL{}
	for _, p := range paths {
		byPath[p] = store.GetACLs(p)
	}
	if decision, decided := decideByACL(ctx, paths, perm, byPath); decided {
		return decision
	}

	return decideByRBAC(ctx, perm)
}

func decideByACL(ctx AuthContext, orderedPaths []string, perm Permission, byPath map[string][]ACL) (bool, bool) {
	for _, p := range orderedPaths {
		acls := byPath[p]
		if len(acls) == 0 {
			continue
		}

		// 1) explicit user ACL precedence
		for _, acl := range acls {
			if acl.PrincipalID != "user:"+ctx.UserID {
				continue
			}
			if allow, ok := acl.Permissions[perm]; ok {
				return allow, true
			}
		}

		// 2) role ACL precedence (deny overrides allow at this level)
		hasRoleAllow := false
		for _, acl := range acls {
			for _, r := range ctx.Roles {
				if acl.PrincipalID != "role:"+r {
					continue
				}
				if allow, ok := acl.Permissions[perm]; ok {
					if !allow {
						return false, true
					}
					hasRoleAllow = true
				}
			}
		}
		if hasRoleAllow {
			return true, true
		}
	}
	return false, false
}

func decideByRBAC(ctx AuthContext, perm Permission) bool {
	for _, r := range ctx.Roles {
		if Roles[r][perm] {
			return true
		}
	}
	return false
}
