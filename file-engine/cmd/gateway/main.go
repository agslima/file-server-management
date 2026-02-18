package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	pb "github.com/example/file-engine/pkg/generated"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// After generating the code, the following function will be available:
	// pb.RegisterFileEngineHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
	// Use the generated pkg to register handlers.
	if err := pb.RegisterFileEngineHandlerFromEndpoint(ctx, mux, "localhost:50051", opts); err != nil {
		return err
	}

	log.Println("gRPC-Gateway listening on :8080")
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return srv.ListenAndServe()
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
