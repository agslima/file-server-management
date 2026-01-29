package authz

import "github.com/example/file-engine/internal/auth"

// MethodPermission maps gRPC full method name -> required permission.
// Keep this list in sync with the proto/handlers so new RPCs are authorized.
var MethodPermission = map[string]auth.Permission{
	// Folder and object operations.
	"/fileengine.FileEngine/CreateFolder":   auth.PermWrite,
	"/fileengine.FileEngine/ListObjects":    auth.PermList,
	"/fileengine.FileEngine/UploadObject":   auth.PermWrite,
	"/fileengine.FileEngine/DownloadObject": auth.PermRead,

	// Async task and upload flows.
	"/fileengine.FileEngine/InitiateUpload": auth.PermWrite,
	"/fileengine.FileEngine/CompleteUpload": auth.PermWrite,
	"/fileengine.FileEngine/GetTaskStatus":  auth.PermRead,
	"/fileengine.FileEngine/GetTask":        auth.PermRead,
}
