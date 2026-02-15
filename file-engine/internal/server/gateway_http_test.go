package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeGatewayServer struct {
	pb.UnimplementedFileEngineServer
}

func (s *fakeGatewayServer) CreateFolder(ctx context.Context, _ *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	return &pb.CreateFolderResponse{TaskId: "task-1", Status: "queued", Message: "scheduled"}, nil
}

func (s *fakeGatewayServer) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	return &pb.TaskStatusResponse{TaskId: req.TaskId, Status: "success", Progress: 100, Message: "done"}, nil
}

func TestGatewayCreateFolderAndGetTaskStatusRoutes(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterFileEngineServer(grpcSrv, &fakeGatewayServer{})
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcSrv.Stop()
		_ = lis.Close()
	})

	mux := runtime.NewServeMux()
	if err := pb.RegisterFileEngineHandlerFromEndpoint(context.Background(), mux, lis.Addr().String(), []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}); err != nil {
		t.Fatalf("register gateway: %v", err)
	}

	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()

	resp, err := http.Post(httpSrv.URL+"/v1/folders", "application/json", bytes.NewBufferString(`{"parentPath":"/tenants/acme","folderName":"reports","requestedBy":"user-1"}`))
	if err != nil {
		t.Fatalf("post /v1/folders: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var createResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp["taskId"] != "task-1" {
		t.Fatalf("unexpected taskId: %v", createResp["taskId"])
	}

	getResp, err := http.Get(httpSrv.URL + "/v1/tasks/task-1")
	if err != nil {
		t.Fatalf("get /v1/tasks: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	var statusResp map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("decode task status response: %v", err)
	}
	if statusResp["taskId"] != "task-1" {
		t.Fatalf("unexpected taskId: %v", statusResp["taskId"])
	}
	if statusResp["status"] != "success" {
		t.Fatalf("unexpected status: %v", statusResp["status"])
	}
}
