package auth
import "path/filepath"
type AuthContext struct{UserID string; Roles []string}
func parents(p string)(r[]string){for{r=append(r,p);if p=="/"{break};p=filepath.Dir(p)};return}
func CanAccess(c AuthContext,p string,perm Permission,s ACLStore)bool{for _,pp:=range parents(p){for _,a:=range s.GetACLs(pp){if a.PrincipalID=="user:"+c.UserID&&a.Permissions[perm]{return true};for _,r:=range c.Roles{if a.PrincipalID=="role:"+r&&a.Permissions[perm]{return true}}}};for _,r:=range c.Roles{if Roles[r][perm]{return true}};return false}
