package handlers

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
	"github.com/example/file-engine/internal/logger"
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

	queue          TaskQueue
	objects        *services.ObjectService
	acl            auth.ACLStore
	tenantResolver auth.TenantResolver
	log            *logger.Logger
}

func NewGRPCHandler(q TaskQueue, obj *services.ObjectService, acl auth.ACLStore, tenantResolver auth.TenantResolver, logg *logger.Logger) *GRPCHandler {
	if tenantResolver == nil {
		tenantResolver = auth.NewDenyAllTenantResolver()
	}
	if logg == nil {
		logg = logger.New("info")
	}
	return &GRPCHandler{queue: q, objects: obj, acl: acl, tenantResolver: tenantResolver, log: logg}
}

// CreateFolder stays async via task queue (worker executes).
func (h *GRPCHandler) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	fullPath, err := authz.ExtractPath(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid path")
	}
	tenantID, err := authz.TenantFromPath(fullPath)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "path must be tenant-scoped")
	}
	allowed, err := h.tenantResolver.UserHasTenant(ctx, authCtx.UserID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "tenant resolution failed")
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "tenant access denied")
	}

	correlationID := correlationIDFromContext(ctx)
	requestedBy := req.RequestedBy
	if requestedBy == "" {
		requestedBy = authCtx.UserID
	}

	parentPath := path.Dir(fullPath)
	if parentPath == "." {
		parentPath = "/"
	}
	folderName := strings.TrimPrefix(fullPath, parentPath+"/")
	if parentPath == "/" {
		folderName = strings.TrimPrefix(fullPath, "/")
	}
	if folderName == "" || strings.Contains(folderName, "/") {
		return nil, status.Error(codes.InvalidArgument, "invalid folder name")
	}

	taskID, err := h.queue.EnqueueCreateFolder(ctx, parentPath, folderName, requestedBy, correlationID)
	if err != nil {
		return nil, err
	}
	h.log.Event("info", "create folder task queued", map[string]any{
		"event":          "task.queued",
		"request":        "create_folder",
		"actor":          requestedBy,
		"tenant":         tenantID,
		"correlation_id": correlationID,
		"task_id":        taskID,
		"parent":         parentPath,
		"folder":         folderName,
	})
	return &pb.CreateFolderResponse{TaskId: taskID, Status: "queued", Message: "Folder creation scheduled"}, nil
}

func (h *GRPCHandler) InitiateUpload(ctx context.Context, _ *pb.InitiateUploadRequest) (*pb.InitiateUploadResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	return nil, status.Error(codes.Unimplemented, "upload initiation not implemented")
}

func (h *GRPCHandler) CompleteUpload(ctx context.Context, _ *pb.CompleteUploadRequest) (*pb.CompleteUploadResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	return nil, status.Error(codes.Unimplemented, "upload completion not implemented")
}

func (h *GRPCHandler) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	requestCorrelationID := correlationIDFromContext(ctx)
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
	correlationID := taskStatus.CorrelationID
	if correlationID == "" {
		correlationID = requestCorrelationID
	}
	h.log.Event("info", "task status queried", map[string]any{
		"event":          "task.status.read",
		"request":        "get_task_status",
		"correlation_id": correlationID,
		"task_id":        taskStatus.TaskID,
		"status":         taskStatus.Status,
	})

	return &pb.TaskStatusResponse{
		TaskId:   taskStatus.TaskID,
		Status:   taskStatus.Status,
		Message:  taskStatus.Message,
		Progress: progress,
	}, nil
}

func (h *GRPCHandler) GetTask(ctx context.Context, _ *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	return nil, status.Error(codes.Unimplemented, "GetTask not implemented")
}

func (h *GRPCHandler) ListObjects(ctx context.Context, req *pb.ListObjectsRequest) (*pb.ListObjectsResponse, error) {
	prefix, err := authz.NormalizePath(req.Prefix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid path")
	}
	items, err := h.objects.List(ctx, prefix)
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
	normalizedPath, err := authz.NormalizePath(req.Path)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid path")
	}
	if err := h.objects.Upload(ctx, normalizedPath, req.Content); err != nil {
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

	r, err := h.objects.Open(stream.Context(), path)
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
		return fallbackCorrelationID()
	}
	if vals := md.Get("x-request-id"); len(vals) > 0 {
		return vals[0]
	}
	if vals := md.Get("x-correlation-id"); len(vals) > 0 {
		return vals[0]
	}
	return fallbackCorrelationID()
}

func fallbackCorrelationID() string {
	return fmt.Sprintf("corr-%d", time.Now().UTC().UnixNano())
}
