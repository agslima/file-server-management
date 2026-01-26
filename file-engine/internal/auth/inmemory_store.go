package auth
type InMemoryACLStore struct{data map[string][]ACL}
func NewInMemoryACLStore()*InMemoryACLStore{return &InMemoryACLStore{data:map[string][]ACL{}}}
func(s*InMemoryACLStore)GetACLs(p string)[]ACL{return s.data[p]}
func(s*InMemoryACLStore)SetACL(a ACL)error{s.data[a.Path]=append(s.data[a.Path],a);return nil}
