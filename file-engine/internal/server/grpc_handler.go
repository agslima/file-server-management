package handlers

import (
    "context"
    "github.com/google/uuid"

    pb "github.com/example/file-engine/pkg/generated"
    "github.com/example/file-engine/internal/services"
)

type GRPCHandler struct {
    pb.UnimplementedFileEngineServer
    service *services.FileService
}

func NewGRPCHandler(s *services.FileService) *GRPCHandler {
    return &GRPCHandler{service: s}
}

func (h *GRPCHandler) CreateFolder(
    ctx context.Context,
    req *pb.CreateFolderRequest,
) (*pb.CreateFolderResponse, error) {

    taskID := uuid.New().String()

    err := h.service.QueueCreateFolder(
        ctx,
        taskID,
        req.ParentPath,
        req.FolderName,
        req.RequestedBy,
    )

    if err != nil {
        return nil, err
    }

    return &pb.CreateFolderResponse{
        TaskId: taskID,
        Status: "queued",
        Message: "Folder creation scheduled",
    }, nil
}
