package server

import (
    "net"

    pb "github.com/example/file-engine/pkg/generated"
    "github.com/example/file-engine/internal/handlers"
    "google.golang.org/grpc"
)

type GRPCServer struct {
    addr    string
    handler *handlers.GRPCHandler
}

func NewGRPCServer(addr string, h *handlers.GRPCHandler) *GRPCServer {
    return &GRPCServer{addr: addr, handler: h}
}

func (s *GRPCServer) Start() error {
    lis, err := net.Listen("tcp", s.addr)
    if err != nil {
        return err
    }

    grpcServer := grpc.NewServer()
    pb.RegisterFileEngineServer(grpcServer, s.handler)

    return grpcServer.Serve(lis)
}
