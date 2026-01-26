package auth
type Permission string
const (
 PermRead Permission = "read"
 PermWrite Permission = "write"
 PermDelete Permission = "delete"
 PermList Permission = "list"
)
