package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/example/file-engine/pkg/generated"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type fakeGatewayServer struct {
	pb.UnimplementedFileEngineServer
}

func (s *fakeGatewayServer) CreateFolder(_ context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	if req.ParentPath == "/tenants/beta/projects" {
		denied, err := status.New(codes.PermissionDenied, "tenant access denied").WithDetails(&errdetails.ErrorInfo{
			Reason:   "tenant_mapping_denied",
			Domain:   "file-engine.authz",
			Metadata: map[string]string{"tenant_id": "beta"},
		})
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, "tenant access denied")
		}
		return nil, denied.Err()
	}
	return &pb.CreateFolderResponse{TaskId: "task-1", Status: "queued", Message: "scheduled"}, nil
}

func (s *fakeGatewayServer) GetTaskStatus(_ context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
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

	mux := runtime.NewServeMux(runtime.WithErrorHandler(gatewayErrorHandler))
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
	defer func() { _ = resp.Body.Close() }()
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
	defer func() { _ = getResp.Body.Close() }()
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

func TestGatewayAuthzDenyResponseShape(t *testing.T) {
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

	mux := runtime.NewServeMux(runtime.WithErrorHandler(gatewayErrorHandler))
	if err := pb.RegisterFileEngineHandlerFromEndpoint(context.Background(), mux, lis.Addr().String(), []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}); err != nil {
		t.Fatalf("register gateway: %v", err)
	}

	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()

	req, err := http.NewRequest(http.MethodPost, httpSrv.URL+"/v1/folders", bytes.NewBufferString(`{"parentPath":"/tenants/beta/projects","folderName":"reports","requestedBy":"user-1"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request denied create folder: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	var body struct {
		Error struct {
			Code          string `json:"code"`
			Reason        string `json:"reason"`
			TenantID      string `json:"tenant_id"`
			RequestID     string `json:"request_id"`
			CorrelationID string `json:"correlation_id"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode denied response: %v", err)
	}
	if body.Error.Code != "AUTHZ_DENY" {
		t.Fatalf("unexpected code: %s", body.Error.Code)
	}
	if body.Error.Reason != "tenant_mapping_denied" {
		t.Fatalf("unexpected reason: %s", body.Error.Reason)
	}
	if body.Error.TenantID != "beta" {
		t.Fatalf("unexpected tenant id: %s", body.Error.TenantID)
	}
	if body.Error.RequestID != "req-123" || body.Error.CorrelationID != "req-123" {
		t.Fatalf("request/correlation IDs should match request id header, got request_id=%q correlation_id=%q", body.Error.RequestID, body.Error.CorrelationID)
	}
}
