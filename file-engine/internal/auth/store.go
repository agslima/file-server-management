package auth

type ACLStore interface {
    GetACLs(path string) []ACL
    SetACL(acl ACL) error
}
