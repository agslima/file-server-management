package auth
var Roles = map[string]map[Permission]bool{
 "admin": {PermRead:true,PermWrite:true,PermDelete:true,PermList:true},
}
