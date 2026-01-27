package main

import (
    "context"
    "flag"
    "log"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
    pb "github.com/example/file-engine/pkg/generated"
)

func run() error {
    ctx := context.Background()
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    mux := runtime.NewServeMux()
    opts := []grpc.DialOption{grpc.WithInsecure()}

    // After generating the code, the following function will be available:
    // pb.RegisterFileEngineHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
    // For development, you can use the generated pkg to register handlers.
    if err := pb.RegisterFileEngineHandlerFromEndpoint(ctx, mux, "localhost:50051", opts); err != nil {
        return err
    }

    log.Println("gRPC-Gateway listening on :8080")
    return http.ListenAndServe(":8080", mux)
}

func main() {
    flag.Parse()
    if err := run(); err != nil {
        log.Fatal(err)
    }
}
