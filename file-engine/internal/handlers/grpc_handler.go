package handlers

import (
    "context"

    pb "github.com/example/file-engine/pkg/generated"
)

type GRPCHandler struct{}

func NewGRPCHandler() *GRPCHandler { return &GRPCHandler{} }

func (h *GRPCHandler) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
    return &pb.CreateFolderResponse{TaskId: "tsk_stub", Status: "queued", Message: "queued"}, nil
}