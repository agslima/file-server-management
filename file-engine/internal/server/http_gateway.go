package server

import (
    "context"
    "net/http"

    pb "github.com/example/file-engine/pkg/generated"
    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
)

type HTTPServer struct {
    addr      string
    grpcAddr string
}

func NewHTTPServer(addr, grpcAddr string) *HTTPServer {
    return &HTTPServer{addr: addr, grpcAddr: grpcAddr}
}

func (h *HTTPServer) Start() error {
    ctx := context.Background()
    mux := runtime.NewServeMux()

    err := pb.RegisterFileEngineHandlerFromEndpoint(
        ctx,
        mux,
        h.grpcAddr,
        []grpc.DialOption{grpc.WithInsecure()},
    )
    if err != nil {
        return err
    }

    return http.ListenAndServe(h.addr, mux)
}
