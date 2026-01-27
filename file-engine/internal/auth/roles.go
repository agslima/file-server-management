package auth

var Roles = map[string]map[Permission]bool{
    "admin":  {PermRead: true, PermWrite: true, PermDelete: true, PermList: true},
    "editor": {PermRead: true, PermWrite: true, PermList: true},
    "viewer": {PermRead: true, PermList: true},
}
