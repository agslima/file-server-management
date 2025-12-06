package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    // pb "github.com/example/file-engine/proto" // uncomment after generating protobuf
    "google.golang.org/grpc"
)

// Simple in-memory task store (stub)
type Task struct {
    ID      string `json:"id"`
    Status  string `json:"status"`
    Message string `json:"message"`
    Created time.Time `json:"created"`
}

var (
    tasks   = map[string]*Task{}
    tasksMu sync.RWMutex
)

func main() {
    // Start HTTP server
    router := mux.NewRouter()

    router.HandleFunc("/health", healthHandler).Methods("GET")
    router.HandleFunc("/create-folder", createFolderHandler).Methods("POST") // simple HTTP endpoint
    router.HandleFunc("/uploads/initiate", uploadsInitiateHandler).Methods("POST")
    router.HandleFunc("/uploads/complete", uploadsCompleteHandler).Methods("POST")
    router.HandleFunc("/tasks/{id}", taskStatusHandler).Methods("GET")

    srv := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    // Start gRPC server in goroutine (proto stubs must be generated)
    go func() {
        lis, err := net.Listen("tcp", ":50051")
        if err != nil {
            log.Fatalf("failed to listen: %v", err)
        }
        grpcServer := grpc.NewServer()
        // Register gRPC services here after generating pb.go from proto.
        // pb.RegisterFileEngineServer(grpcServer, &grpcServerImpl{})
        log.Println("gRPC server listening on :50051 (note: proto stubs must be generated for gRPC to work)")
        if err := grpcServer.Serve(lis); err != nil {
            log.Fatalf("gRPC serve error: %v", err)
        }
    }()

    // Start HTTP server
    go func() {
        log.Println("HTTP server listening on :8080")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s\n", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down servers...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server Shutdown Failed:%+v", err)
    }
    log.Println("Server exited properly")
}

// Handlers (stubs)
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}

type CreateFolderReq struct {
    ParentPath string `json:"parent_path"`
    Name       string `json:"name"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type CreateFolderResp struct {
    Status  string `json:"status"`
    TaskID  string `json:"task_id"`
    Message string `json:"message"`
}

func createFolderHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateFolderReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }
    // Basic validation (stub)
    if req.ParentPath == "" || req.Name == "" {
        http.Error(w, "parent_path and name are required", http.StatusBadRequest)
        return
    }
    // Create a task (stub)
    taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
    tasksMu.Lock()
    tasks[taskID] = &Task{ID: taskID, Status: "queued", Message: "Task queued for folder creation", Created: time.Now()}
    tasksMu.Unlock()

    // In a real implementation, enqueue to Redis/Kafka and have a worker create the folder on the remote FS.

    resp := CreateFolderResp{Status: "queued", TaskID: taskID, Message: "Pasta em criação. Verifique o status via /tasks/" + taskID}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(resp)
}

type UploadInitiateReq struct {
    TargetPath string `json:"target_path"`
    Filename   string `json:"filename"`
    Size       int64  `json:"size"`
}

type UploadInitiateResp struct {
    UploadURL string `json:"upload_url"`
    UploadID  string `json:"upload_id"`
}

func uploadsInitiateHandler(w http.ResponseWriter, r *http.Request) {
    var req UploadInitiateReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }
    // Stub: generate presigned URL (in real system, call S3/minio)
    uploadID := fmt.Sprintf("upload-%d", time.Now().UnixNano())
    uploadURL := fmt.Sprintf("https://temp-storage.example.com/upload/%s", uploadID)

    resp := UploadInitiateResp{UploadURL: uploadURL, UploadID: uploadID}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

type UploadCompleteReq struct {
    UploadID string `json:"upload_id"`
}

type UploadCompleteResp struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}

func uploadsCompleteHandler(w http.ResponseWriter, r *http.Request) {
    var req UploadCompleteReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }
    if req.UploadID == "" {
        http.Error(w, "upload_id is required", http.StatusBadRequest)
        return
    }
    // Stub: mark for scan and moving
    taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
    tasksMu.Lock()
    tasks[taskID] = &Task{ID: taskID, Status: "queued", Message: "Arquivo recebido e enviado para scan", Created: time.Now()}
    tasksMu.Unlock()

    resp := UploadCompleteResp{Status: "queued_scan", Message: "Arquivo recebido e enviado para scan. Após aprovação, será movido para o File Server."}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func taskStatusHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    tasksMu.RLock()
    t, ok := tasks[id]
    tasksMu.RUnlock()
    if !ok {
        http.Error(w, "task not found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(t)
}

// --- gRPC server stub (for when proto is generated) ---
// type grpcServerImpl struct{}
// // Implement gRPC methods corresponding to proto definitions here.