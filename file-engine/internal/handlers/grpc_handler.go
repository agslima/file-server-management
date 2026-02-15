package handlers

import (
	"context"
	"io"
	"log"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type TaskQueue interface {
	EnqueueCreateFolder(ctx context.Context, parentPath, folderName, requestedBy, correlationID string) (string, error)
	GetStatus(ctx context.Context, id string) (*redisq.TaskStatus, error)
}

type GRPCHandler struct {
	pb.UnimplementedFileEngineServer

	queue   TaskQueue
	objects *services.ObjectService
	acl     auth.ACLStore
}

func NewGRPCHandler(q TaskQueue, obj *services.ObjectService, acl auth.ACLStore) *GRPCHandler {
	return &GRPCHandler{queue: q, objects: obj, acl: acl}
}

// CreateFolder stays async via task queue (worker executes).
func (h *GRPCHandler) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	correlationID := correlationIDFromContext(ctx)
	requestedBy := req.RequestedBy
	if requestedBy == "" {
		requestedBy = authCtx.UserID
	}

	taskID, err := h.queue.EnqueueCreateFolder(ctx, req.ParentPath, req.FolderName, requestedBy, correlationID)
	if err != nil {
		return nil, err
	}
	log.Printf("request=create_folder actor=%s correlation_id=%s task_id=%s parent=%s folder=%s", requestedBy, correlationID, taskID, req.ParentPath, req.FolderName)
	log.Printf("audit_event=task.queued task_id=%s correlation_id=%s message=%q", taskID, correlationID, "folder creation queued")
	return &pb.CreateFolderResponse{TaskId: taskID, Status: "queued", Message: "Folder creation scheduled"}, nil
}

func (h *GRPCHandler) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	taskStatus, err := h.queue.GetStatus(ctx, req.TaskId)
	if err != nil {
		if err == redisq.ErrTaskNotFound {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Error(codes.Internal, "failed to load task status")
	}

	progress := int32(100)
	if taskStatus.Status == "queued" || taskStatus.Status == "running" {
		progress = 0
	}

	return &pb.TaskStatusResponse{
		TaskId:   taskStatus.TaskID,
		Status:   taskStatus.Status,
		Message:  taskStatus.Message,
		Progress: progress,
	}, nil
}

func (h *GRPCHandler) ListObjects(ctx context.Context, req *pb.ListObjectsRequest) (*pb.ListObjectsResponse, error) {
	items, err := h.objects.List(ctx, req.Prefix)
	if err != nil {
		return nil, err
	}
	out := &pb.ListObjectsResponse{}
	for _, it := range items {
		out.Items = append(out.Items, &pb.ObjectInfo{
			Path:  it.Path,
			Size:  it.Size,
			IsDir: it.IsDir,
		})
	}
	return out, nil
}

func (h *GRPCHandler) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	if err := h.objects.Upload(ctx, req.Path, req.Content); err != nil {
		return nil, err
	}
	return &pb.UploadObjectResponse{Success: true}, nil
}

func (h *GRPCHandler) DownloadObject(req *pb.DownloadObjectRequest, stream pb.FileEngine_DownloadObjectServer) error {
	authCtx, ok := auth.FromContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "missing auth context")
	}
	path, err := authz.ExtractPath(req)
	if err != nil {
		return status.Error(codes.InvalidArgument, "cannot extract path")
	}
	if !auth.CanAccess(authCtx, path, auth.PermRead, h.acl) {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	r, err := h.objects.Open(stream.Context(), req.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	buf := make([]byte, 64*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if err2 := stream.Send(&pb.DownloadChunk{Data: buf[:n]}); err2 != nil {
				return err2
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func correlationIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if vals := md.Get("x-request-id"); len(vals) > 0 {
		return vals[0]
	}
	if vals := md.Get("x-correlation-id"); len(vals) > 0 {
		return vals[0]
	}
	return ""
}
