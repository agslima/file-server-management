package auth

type ACL struct {
    Path        string
    PrincipalID string // user:42 | role:admin | service:xyz
    Permissions map[Permission]bool
}
