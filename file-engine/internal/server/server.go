package server

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "time"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/example/file-engine/pkg/generated"
    "github.com/example/file-engine/internal/auth"
    "github.com/example/file-engine/internal/authz"
    "github.com/example/file-engine/internal/logger"
    "github.com/example/file-engine/internal/storage"
)

type GRPCServer struct {
    Addr string
    Log  *logger.Logger

    Verifier *auth.JWTVerifier
    ACLStore  auth.ACLStore

    Handler pb.FileEngineServer
}

type HTTPServer struct {
    Addr    string
    GRPCAddr string
    Log     *logger.Logger

    Verifier *auth.JWTVerifier
    Storage  storage.Storage

    ACLStore auth.ACLStore
}

func NewGRPCServer(addr string, logg *logger.Logger, verifier *auth.JWTVerifier, store auth.ACLStore, handler pb.FileEngineServer) *GRPCServer {
    return &GRPCServer{
        Addr: addr, Log: logg,
        Verifier: verifier, ACLStore: store,
        Handler: handler,
    }
}

func NewHTTPServer(addr, grpcAddr string, logg *logger.Logger, verifier *auth.JWTVerifier, st storage.Storage, store auth.ACLStore) *HTTPServer {
    return &HTTPServer{Addr: addr, GRPCAddr: grpcAddr, Log: logg, Verifier: verifier, Storage: st, ACLStore: store}
}

func (g *GRPCServer) Start() error {
    lis, err := net.Listen("tcp", g.Addr)
    if err != nil {
        return fmt.Errorf("listen: %w", err)
    }

    srv := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            auth.GRPCAuthInterceptor(g.Verifier),
            authz.GRPCAuthZInterceptor(g.ACLStore),
        ),
    )

    pb.RegisterFileEngineServer(srv, g.Handler)

    g.Log.Infof("gRPC listening on %s", g.Addr)
    return srv.Serve(lis)
}

func (h *HTTPServer) Start() error {
    ctx := context.Background()

    mux := runtime.NewServeMux()

    conn, err := grpc.DialContext(
        ctx,
        h.GRPCAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return fmt.Errorf("dial grpc: %w", err)
    }

    // Register gateway handlers
    if err := pb.RegisterFileEngineHandler(ctx, mux, conn); err != nil {
        return fmt.Errorf("register gateway: %w", err)
    }

    root := http.NewServeMux()
    // Raw download endpoint (REST-friendly): streams bytes directly.
    root.HandleFunc("/v1/objects:download", h.handleDownload)

    // Gateway endpoints (ListObjects, UploadObject, CreateFolder, TaskStatus, etc.)
    root.Handle("/", auth.HTTPAuthMiddleware(h.Verifier, mux))

    srv := &http.Server{
        Addr:              h.Addr,
        Handler:           root,
        ReadHeaderTimeout: 10 * time.Second,
    }

    h.Log.Infof("HTTP listening on %s (proxying to %s)", h.Addr, h.GRPCAddr)
    return srv.ListenAndServe()
}
