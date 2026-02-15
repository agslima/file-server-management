package generated

import (
	"context"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Stub gRPC types for local builds. Replace with real generated code.

type CreateFolderRequest struct {
	ParentPath  string
	FolderName  string
	RequestedBy string
}

type CreateFolderResponse struct {
	TaskId  string
	Status  string
	Message string
}

type InitiateUploadRequest struct {
	TargetPath string
	Filename   string
	Size       int64
	Mime       string
}

type InitiateUploadResponse struct {
	UploadId  string
	UploadUrl string
}

type CompleteUploadRequest struct {
	UploadId string
}

type CompleteUploadResponse struct {
	TaskId string
	Status string
}

type TaskStatusRequest struct {
	TaskId string
}

type TaskStatusResponse struct {
	TaskId   string
	Status   string
	Progress int32
	Message  string
}

type ObjectInfo struct {
	Path  string
	Size  int64
	IsDir bool
}

type ListObjectsRequest struct {
	Prefix string
}

type ListObjectsResponse struct {
	Items []*ObjectInfo
}

type UploadObjectRequest struct {
	Path    string
	Content []byte
}

type UploadObjectResponse struct {
	Success bool
}

type DownloadObjectRequest struct {
	Path string
}

type DownloadChunk struct {
	Data []byte
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

type FileEngine_DownloadObjectServer interface {
	Send(*DownloadChunk) error
	Context() context.Context
}

type FileEngineServer interface {
	CreateFolder(context.Context, *CreateFolderRequest) (*CreateFolderResponse, error)
	InitiateUpload(context.Context, *InitiateUploadRequest) (*InitiateUploadResponse, error)
	CompleteUpload(context.Context, *CompleteUploadRequest) (*CompleteUploadResponse, error)
	GetTaskStatus(context.Context, *TaskStatusRequest) (*TaskStatusResponse, error)
	ListObjects(context.Context, *ListObjectsRequest) (*ListObjectsResponse, error)
	UploadObject(context.Context, *UploadObjectRequest) (*UploadObjectResponse, error)
	DownloadObject(*DownloadObjectRequest, FileEngine_DownloadObjectServer) error
	GetTask(context.Context, *GetTaskRequest) (*GetTaskResponse, error)
}

type UnimplementedFileEngineServer struct{}

func (UnimplementedFileEngineServer) CreateFolder(context.Context, *CreateFolderRequest) (*CreateFolderResponse, error) {
	return nil, io.EOF
}
func (UnimplementedFileEngineServer) InitiateUpload(context.Context, *InitiateUploadRequest) (*InitiateUploadResponse, error) {
	return nil, io.EOF
}
func (UnimplementedFileEngineServer) CompleteUpload(context.Context, *CompleteUploadRequest) (*CompleteUploadResponse, error) {
	return nil, io.EOF
}
func (UnimplementedFileEngineServer) GetTaskStatus(context.Context, *TaskStatusRequest) (*TaskStatusResponse, error) {
	return nil, io.EOF
}
func (UnimplementedFileEngineServer) ListObjects(context.Context, *ListObjectsRequest) (*ListObjectsResponse, error) {
	return nil, io.EOF
}
func (UnimplementedFileEngineServer) UploadObject(context.Context, *UploadObjectRequest) (*UploadObjectResponse, error) {
	return nil, io.EOF
}
func (UnimplementedFileEngineServer) DownloadObject(*DownloadObjectRequest, FileEngine_DownloadObjectServer) error {
	return io.EOF
}
func (UnimplementedFileEngineServer) GetTask(context.Context, *GetTaskRequest) (*GetTaskResponse, error) {
	return nil, io.EOF
}

func RegisterFileEngineServer(_ *grpc.Server, _ FileEngineServer) {}

func RegisterFileEngineHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}

func RegisterFileEngineHandlerFromEndpoint(_ context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
	return nil
}
