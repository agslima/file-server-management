package authz

import "github.com/example/file-engine/internal/auth"

// MethodPermission maps gRPC full method name -> required permission.
var MethodPermission = map[string]auth.Permission{
	"/fileengine.FileEngine/CreateFolder":   auth.PermWrite,
	"/fileengine.FileEngine/InitiateUpload": auth.PermWrite,
	"/fileengine.FileEngine/CompleteUpload": auth.PermWrite,
	"/fileengine.FileEngine/GetTaskStatus":  auth.PermRead,
	"/fileengine.FileEngine/GetTask":        auth.PermRead,
	"/fileengine.FileEngine/ListObjects":    auth.PermList,
	"/fileengine.FileEngine/UploadObject":   auth.PermWrite,
	"/fileengine.FileEngine/DownloadObject": auth.PermRead,
}
