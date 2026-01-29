package handlers

import (
    "context"
    "io"

    pb "github.com/example/file-engine/pkg/generated"
    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/services"
)

type GRPCHandler struct {
    pb.UnimplementedFileEngineServer

    queue   *redisq.RedisQueue
    objects *services.ObjectService
}

func NewGRPCHandler(q *redisq.RedisQueue, obj *services.ObjectService) *GRPCHandler {
    return &GRPCHandler{queue: q, objects: obj}
}

// CreateFolder stays async via task queue (worker executes).
func (h *GRPCHandler) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
    // minimal enqueue (task payload shape in your queue package)
    taskID, err := h.queue.EnqueueCreateFolder(ctx, req.ParentPath, req.FolderName, req.RequestedBy)
    if err != nil {
        return nil, err
    }
    return &pb.CreateFolderResponse{TaskId: taskID, Status: "queued", Message: "Folder creation scheduled"}, nil
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
