package main

import (
    "context"
    "flag"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    "google.golang.org/grpc"
    pb "github.com/example/file-engine/grpc/pkg"
)

type server struct {
    pb.UnimplementedFileEngineServer
}

func (s *server) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
    // stub: create task and return task id
    return &pb.CreateFolderResponse{TaskId: "tsk_stub_1", Status: "queued", Message: "queued"}, nil
}
func (s *server) InitiateUpload(ctx context.Context, req *pb.InitiateUploadRequest) (*pb.InitiateUploadResponse, error) {
    return &pb.InitiateUploadResponse{UploadId: "upl_stub_1", UploadUrl: "https://storage.example/upload/upl_stub_1"}, nil
}
func (s *server) CompleteUpload(ctx context.Context, req *pb.CompleteUploadRequest) (*pb.CompleteUploadResponse, error) {
    return &pb.CompleteUploadResponse{TaskId: "tsk_upload_1", Status: "queued"}, nil
}
func (s *server) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
    return &pb.GetTaskResponse{TaskId: req.TaskId, Status: "completed", Progress: 100, Message: "done"}, nil
}

func main() {
    var grpcAddr = flag.String("grpc", ":50051", "gRPC listen address")
    flag.Parse()

    lis, err := net.Listen("tcp", *grpcAddr)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    srv := grpc.NewServer()
    pb.RegisterFileEngineServer(srv, &server{})

    go func() {
        log.Printf("gRPC server listening on %s", *grpcAddr)
        if err := srv.Serve(lis); err != nil {
            log.Fatalf("gRPC serve error: %v", err)
        }
    }()

    // Wait for signal to exit
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
    <-ch
    log.Println("shutting down")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = ctx
    srv.GracefulStop()
}