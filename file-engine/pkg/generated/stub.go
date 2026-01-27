package generated

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Stub gRPC types for local builds. Replace with real generated code.

type CreateFolderRequest struct {
	Path       string
	ParentPath string
	Name       string
	Metadata   map[string]string
}

type CreateFolderResponse struct {
	Ok      bool
	TaskId  string
	Status  string
	Message string
}

type InitiateUploadRequest struct {
	TargetPath string
	Filename   string
	Size       int64
}

type InitiateUploadResponse struct {
	UploadId  string
	UploadUrl string
}

type CompleteUploadRequest struct {
	UploadId   string
	TargetPath string
}

type CompleteUploadResponse struct {
	Ok     bool
	TaskId string
	Status string
}

type GetTaskRequest struct {
	TaskId string
}

type GetTaskResponse struct {
	TaskId   string
	Status   string
	Progress int32
	Message  string
	Details  string
}

type FileEngineServer interface {
	CreateFolder(context.Context, *CreateFolderRequest) (*CreateFolderResponse, error)
	InitiateUpload(context.Context, *InitiateUploadRequest) (*InitiateUploadResponse, error)
	CompleteUpload(context.Context, *CompleteUploadRequest) (*CompleteUploadResponse, error)
	GetTask(context.Context, *GetTaskRequest) (*GetTaskResponse, error)
}

type UnimplementedFileEngineServer struct{}

func RegisterFileEngineServer(_ *grpc.Server, _ FileEngineServer) {}

func RegisterFileEngineHandlerFromEndpoint(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
	return nil
}
