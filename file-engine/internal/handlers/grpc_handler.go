package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TaskQueue interface {
	EnqueueCreateFolder(ctx context.Context, parentPath, folderName, requestedBy, correlationID string) (string, error)
	GetStatus(ctx context.Context, id string) (*redisq.TaskStatus, error)
}

type GRPCHandler struct {
	pb.UnimplementedFileEngineServer

	queue           TaskQueue
	objects         *services.ObjectService
	uploads         *services.UploadService
	acl             auth.ACLStore
	tenantResolver  auth.TenantResolver
	log             *logger.Logger
	auditor         AuditEmitter
	rateMu          sync.Mutex
	enqueueByTenant map[string]int
	enqueueByActor  map[string]int
	enqueueReset    time.Time
}

type AuditEmitter interface {
	EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string, metadata ...map[string]string)
}

type noopAuditEmitter struct{}

func (noopAuditEmitter) EmitTaskEvent(context.Context, string, string, string, string, ...map[string]string) {
}

func NewGRPCHandler(q TaskQueue, obj *services.ObjectService, uploads *services.UploadService, acl auth.ACLStore, tenantResolver auth.TenantResolver, logg *logger.Logger, auditor AuditEmitter) *GRPCHandler {
	if tenantResolver == nil {
		tenantResolver = auth.NewDenyAllTenantResolver()
	}
	if logg == nil {
		logg = logger.New("info")
	}
	if auditor == nil {
		auditor = noopAuditEmitter{}
	}
	return &GRPCHandler{queue: q, objects: obj, uploads: uploads, acl: acl, tenantResolver: tenantResolver, log: logg, auditor: auditor, enqueueByTenant: map[string]int{}, enqueueByActor: map[string]int{}, enqueueReset: time.Now().Add(time.Second)}
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
		h.auditor.EmitTaskEvent(ctx, "auth.decision.error", "", correlationIDFromContext(ctx), "tenant resolution failed")
		return nil, status.Error(codes.Internal, "tenant resolution failed")
	}
	if !allowed {
		h.auditor.EmitTaskEvent(ctx, "auth.decision.denied", "", correlationIDFromContext(ctx), "tenant access denied")
		return nil, tenantMappingDeniedStatus(correlationIDFromContext(ctx), tenantID).Err()
	}
	h.auditor.EmitTaskEvent(ctx, "auth.decision.allowed", "", correlationIDFromContext(ctx), "tenant access allowed")

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

	if !h.allowEnqueue(tenantID, requestedBy) {
		observability.DefaultMetrics.IncRateLimitBlock("enqueue", "tenant_or_actor_rps")
		h.auditor.EmitTaskEvent(ctx, "task.enqueue.throttled", "", correlationID, "enqueue rate limit exceeded")
		return nil, status.Error(codes.ResourceExhausted, "enqueue rate limit exceeded")
	}
	taskID, err := h.queue.EnqueueCreateFolder(ctx, parentPath, folderName, requestedBy, correlationID)
	if err != nil {
		if errors.Is(err, redisq.ErrQueueUnavailable) {
			observability.DefaultMetrics.IncRateLimitBlock("queue", "backpressure")
			h.auditor.EmitTaskEvent(ctx, "task.enqueue.rejected", "", correlationID, "queue unavailable")
			return nil, status.Error(codes.Unavailable, "queue unavailable")
		}
		return nil, err
	}
	h.auditor.EmitTaskEvent(ctx, "task.enqueued", taskID, correlationID, "create_folder queued")
	h.auditor.EmitTaskEvent(ctx, "folder.mutation.requested", taskID, correlationID, "create_folder requested")
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
		if errors.Is(err, redisq.ErrTaskNotFound) {
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
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	prefix, err := authz.NormalizePath(req.Prefix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid path")
	}
	if err := h.ensureTenantAccess(ctx, authCtx, prefix, auth.PermList); err != nil {
		return nil, err
	}
	tenantID, err := authz.TenantFromPath(prefix)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "path must be tenant-scoped")
	}

	items, err := h.objects.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	out := &pb.ListObjectsResponse{}
	for _, it := range items {
		var modifiedAt *timestamppb.Timestamp
		if !it.ModifiedAt.IsZero() {
			modifiedAt = timestamppb.New(it.ModifiedAt)
		}
		var createdAt *timestamppb.Timestamp
		if !it.CreatedAt.IsZero() {
			createdAt = timestamppb.New(it.CreatedAt)
		}
		out.Items = append(out.Items, &pb.ObjectInfo{
			Path:       it.Path,
			Size:       it.Size,
			IsDir:      it.IsDir,
			ModifiedAt: modifiedAt,
			CreatedAt:  createdAt,
			Owner:      it.Owner,
			Group:      it.Group,
		})
	}
	h.emitReadAudit(ctx, "object.list", tenantID, authCtx, "success", prefix)
	return out, nil
}

func (h *GRPCHandler) UploadObject(ctx context.Context, req *pb.UploadObjectRequest) (*pb.UploadObjectResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	normalizedPath, err := authz.NormalizePath(req.Path)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid path")
	}
	if err := h.ensureTenantAccess(ctx, authCtx, normalizedPath, auth.PermWrite); err != nil {
		return nil, err
	}
	h.auditor.EmitTaskEvent(ctx, "upload.started", "", correlationIDFromContext(ctx), normalizedPath)
	if h.uploads == nil {
		if err := h.objects.Upload(ctx, normalizedPath, req.Content); err != nil {
			return nil, err
		}
	} else {
		idempotencyKey := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-idempotency-key"); len(vals) > 0 {
				idempotencyKey = vals[0]
			}
		}
		if _, err := h.uploads.UploadStream(ctx, normalizedPath, bytes.NewReader(req.Content), idempotencyKey); err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}
	h.auditor.EmitTaskEvent(ctx, "upload.completed", "", correlationIDFromContext(ctx), normalizedPath)
	return &pb.UploadObjectResponse{Success: true}, nil
}

func (h *GRPCHandler) DownloadObject(req *pb.DownloadObjectRequest, stream pb.FileEngine_DownloadObjectServer) error {
	authCtx, ok := auth.FromContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "missing auth context")
	}
	objectPath, err := authz.ExtractPath(req)
	if err != nil {
		return status.Error(codes.InvalidArgument, "cannot extract path")
	}
	if err := h.ensureTenantAccess(stream.Context(), authCtx, objectPath, auth.PermRead); err != nil {
		return err
	}
	tenantID, err := authz.TenantFromPath(objectPath)
	if err != nil {
		return status.Error(codes.InvalidArgument, "path must be tenant-scoped")
	}
	h.emitReadAudit(stream.Context(), "object.read", tenantID, authCtx, "success", objectPath)

	r, err := h.objects.Open(stream.Context(), objectPath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	buf := make([]byte, 64*1024)
	hashr := sha256.New()
	for {
		n, err := r.Read(buf)
		if n > 0 {
			_, _ = hashr.Write(buf[:n])
			if err2 := stream.Send(&pb.DownloadChunk{Data: buf[:n]}); err2 != nil {
				return err2
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if h.uploads != nil {
		if err := h.uploads.VerifyIntegrityHash(objectPath, hex.EncodeToString(hashr.Sum(nil))); err != nil {
			return status.Error(codes.DataLoss, "integrity validation failed")
		}
	}
	h.emitReadAudit(stream.Context(), "object.download", tenantID, authCtx, "success", objectPath)
	return nil
}

func (h *GRPCHandler) ensureTenantAccess(ctx context.Context, authCtx auth.AuthContext, p string, perm auth.Permission) error {
	tenantID, err := authz.TenantFromPath(p)
	if err != nil {
		return status.Error(codes.InvalidArgument, "path must be tenant-scoped")
	}
	allowed, err := h.tenantResolver.UserHasTenant(ctx, authCtx.UserID, tenantID)
	if err != nil {
		h.auditor.EmitTaskEvent(ctx, "auth.decision.error", "", correlationIDFromContext(ctx), "tenant resolution failed")
		observability.DefaultMetrics.ObserveAuthzDecision(false, "tenant_resolver_error")
		return status.Error(codes.Internal, "tenant resolution failed")
	}
	if !allowed {
		h.auditor.EmitTaskEvent(ctx, "auth.decision.denied", "", correlationIDFromContext(ctx), "tenant access denied")
		observability.DefaultMetrics.ObserveAuthzDecision(false, "tenant_membership")
		return tenantMappingDeniedStatus(correlationIDFromContext(ctx), tenantID).Err()
	}
	if !auth.CanAccess(authCtx, p, perm, h.acl) {
		h.auditor.EmitTaskEvent(ctx, "auth.decision.denied", "", correlationIDFromContext(ctx), "access denied")
		observability.DefaultMetrics.ObserveAuthzDecision(false, "acl")
		return status.Error(codes.PermissionDenied, "access denied")
	}
	h.auditor.EmitTaskEvent(ctx, "auth.decision.allowed", "", correlationIDFromContext(ctx), "access allowed")
	observability.DefaultMetrics.ObserveAuthzDecision(true, "ok")
	return nil
}

func tenantMappingDeniedStatus(correlationID, tenantID string) *status.Status {
	st := status.New(codes.PermissionDenied, "tenant access denied")
	withDetails, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "tenant_mapping_denied",
		Domain: "file-engine.authz",
		Metadata: map[string]string{
			"tenant_id":      tenantID,
			"correlation_id": correlationID,
			"request_id":     correlationID,
		},
	})
	if err != nil {
		return st
	}
	return withDetails
}

func (h *GRPCHandler) emitReadAudit(ctx context.Context, action, tenantID string, authCtx auth.AuthContext, result, resourcePath string) {
	groups := strings.Join(authCtx.Groups, ",")
	h.auditor.EmitTaskEvent(ctx, action, "", correlationIDFromContext(ctx), resourcePath, map[string]string{
		"tenant_id":       tenantID,
		"actor_id":        authCtx.EffectiveActorID(),
		"actor_subject":   authCtx.UserID,
		"actor_email":     authCtx.Email,
		"actor_groups":    groups,
		"actor_roles":     strings.Join(authCtx.Roles, ","),
		"identity_source": "jwt",
		"action":          action,
		"result":          result,
	})
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

func (h *GRPCHandler) allowEnqueue(tenantID, actorID string) bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	now := time.Now()
	if now.After(h.enqueueReset) {
		h.enqueueByTenant = map[string]int{}
		h.enqueueByActor = map[string]int{}
		h.enqueueReset = now.Add(time.Second)
	}
	h.enqueueByTenant[tenantID]++
	h.enqueueByActor[actorID]++
	return h.enqueueByTenant[tenantID] <= 30 && h.enqueueByActor[actorID] <= 15
}
