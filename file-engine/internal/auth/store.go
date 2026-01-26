package auth
type ACLStore interface{GetACLs(string) []ACL; SetACL(ACL) error}
