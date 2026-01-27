package auth

import "path/filepath"

type AuthContext struct {
    UserID string
    Roles  []string
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

func CanAccess(ctx AuthContext, path string, perm Permission, store ACLStore) bool {
    paths := parentPaths(path)

    if bs, ok := store.(ACLBatchStore); ok {
        acls, err := bs.GetACLsForPaths(paths)
        if err == nil {
            byPath := map[string][]ACL{}
            for _, a := range acls {
                byPath[a.Path] = append(byPath[a.Path], a)
            }
            for _, p := range paths {
                for _, acl := range byPath[p] {
                    if acl.PrincipalID == "user:"+ctx.UserID && acl.Permissions[perm] {
                        return true
                    }
                    for _, r := range ctx.Roles {
                        if acl.PrincipalID == "role:"+r && acl.Permissions[perm] {
                            return true
                        }
                    }
                }
            }
            for _, r := range ctx.Roles {
                if Roles[r][perm] {
                    return true
                }
            }
            return false
        }
    }

    for _, p := range paths {
        for _, acl := range store.GetACLs(p) {
            if acl.PrincipalID == "user:"+ctx.UserID && acl.Permissions[perm] {
                return true
            }
            for _, r := range ctx.Roles {
                if acl.PrincipalID == "role:"+r && acl.Permissions[perm] {
                    return true
                }
            }
        }
    }

    for _, r := range ctx.Roles {
        if Roles[r][perm] {
            return true
        }
    }
    return false
}
