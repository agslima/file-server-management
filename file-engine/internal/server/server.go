package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/file-engine/internal/observability"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
	"github.com/example/file-engine/internal/identity"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/services"
	"github.com/example/file-engine/internal/storage"
	pb "github.com/example/file-engine/pkg/generated"
)

type GRPCServer struct {
	Addr string
	Log  *logger.Logger

	Verifier *auth.JWTVerifier
	ACLStore auth.ACLStore

	Handler pb.FileEngineServer
}

type readinessCheck struct {
	name string
	fn   func(context.Context) error
}

type readinessCheckResult struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

type HTTPServer struct {
	Addr     string
	GRPCAddr string
	Log      *logger.Logger

	Verifier *auth.JWTVerifier
	Storage  storage.Storage

	ACLStore      auth.ACLStore
	Uploads       *services.UploadService
	Tenants       auth.TenantResolver
	Identity      *identity.Store
	UploadAuditor UploadAuditEmitter

	ReadyChecks []readinessCheck

	MaxUploadBytes     int64
	UploadTimeout      time.Duration
	sem                chan struct{}
	rateMu             sync.Mutex
	rateByTenant       map[string]int
	rateByActor        map[string]int
	rateReset          time.Time
	concurrentByTenant map[string]int
	concurrentByActor  map[string]int
}

func NewGRPCServer(addr string, logg *logger.Logger, verifier *auth.JWTVerifier, store auth.ACLStore, handler pb.FileEngineServer) *GRPCServer {
	return &GRPCServer{Addr: addr, Log: logg, Verifier: verifier, ACLStore: store, Handler: handler}
}

func NewHTTPServer(addr, grpcAddr string, logg *logger.Logger, verifier *auth.JWTVerifier, st storage.Storage, store auth.ACLStore, uploads *services.UploadService, tenants auth.TenantResolver) *HTTPServer {
	if tenants == nil {
		tenants = auth.NewDenyAllTenantResolver()
	}
	return &HTTPServer{
		Addr: addr, GRPCAddr: grpcAddr, Log: logg, Verifier: verifier, Storage: st, ACLStore: store, Uploads: uploads, Tenants: tenants,
		MaxUploadBytes: 20 * 1024 * 1024, UploadTimeout: 30 * time.Second,
		sem: make(chan struct{}, 8), rateByTenant: map[string]int{}, rateByActor: map[string]int{}, concurrentByTenant: map[string]int{}, concurrentByActor: map[string]int{}, rateReset: time.Now().Add(time.Minute),
	}
}

func (h *HTTPServer) AddReadyCheck(name string, check func(context.Context) error) {
	if check == nil {
		return
	}
	h.ReadyChecks = append(h.ReadyChecks, readinessCheck{name: name, fn: check})
}

func (g *GRPCServer) Start() error {
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", g.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	metricsUnary := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		grpcCode := codes.OK.String()
		if err != nil {
			grpcCode = status.Code(err).String()
		}
		observability.DefaultMetrics.ObserveGRPCRequest(info.FullMethod, grpcCode, time.Since(start).Milliseconds())
		return resp, err
	}
	metricsStream := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		grpcCode := codes.OK.String()
		if err != nil {
			grpcCode = status.Code(err).String()
		}
		observability.DefaultMetrics.ObserveGRPCRequest(info.FullMethod, grpcCode, time.Since(start).Milliseconds())
		return err
	}

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			metricsUnary,
			auth.GRPCAuthInterceptor(g.Verifier),
			authz.GRPCAuthZInterceptor(g.ACLStore),
		),
		grpc.ChainStreamInterceptor(
			metricsStream,
			auth.GRPCStreamAuthInterceptor(g.Verifier),
			authz.GRPCAuthZStreamInterceptor(g.ACLStore),
		),
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterFileEngineServer(srv, g.Handler)
	g.Log.Infof("gRPC listening on %s", g.Addr)
	return srv.Serve(lis)
}

func (h *HTTPServer) Start() error {
	ctx := context.Background()
	mux := runtime.NewServeMux(runtime.WithErrorHandler(gatewayErrorHandler))

	conn, err := grpc.NewClient(h.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial grpc: %w", err)
	}
	if err := pb.RegisterFileEngineHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	root := http.NewServeMux()
	root.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(observability.DefaultMetrics.SnapshotPrometheus()))
	})
	root.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	root.HandleFunc("/readyz", h.handleReadyz)
	root.HandleFunc("/v1/objects:download", h.handleDownload)
	root.HandleFunc("/v1/objects:upload", h.handleUpload)
	root.HandleFunc("/v1/uploads:initiate", h.handleUploadInitiate)
	root.HandleFunc("/v1/uploads/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ":chunk") {
			h.handleUploadChunk(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, ":complete") {
			h.handleUploadComplete(w, r)
			return
		}
		h.writeUploadError(w, r, http.StatusNotFound, "not_found", "upload endpoint not found", "")
	})
	root.HandleFunc("/v1/uploads/complete", h.handleUploadComplete)
	root.HandleFunc("/admin/v1/tenants", h.handleAdminTenants)
	root.HandleFunc("/admin/v1/users", h.handleAdminUsers)
	root.HandleFunc("/admin/v1/memberships", h.handleAdminMemberships)
	root.HandleFunc("/admin/v1/roles", h.handleAdminRoles)
	root.HandleFunc("/admin/v1/access-review", h.handleAccessReview)
	root.HandleFunc("/admin/v1/scan-dlq", h.handleScanDLQ)
	root.HandleFunc("/admin/v1/quarantine:cleanup", h.handleQuarantineCleanup)
	root.HandleFunc("/admin/v1/quarantine:restore", h.handleQuarantineRestore)
	root.HandleFunc("/admin/v1/lifecycle:cleanup", h.handleLifecycleCleanup)
	root.HandleFunc("/admin/v1/governance:delete", h.handleGovernanceDelete)
	root.HandleFunc("/v1/objects:move", h.handleObjectMove)
	root.HandleFunc("/v1/objects:delete", h.handleObjectDelete)
	root.HandleFunc("/admin/v1/governance:effective", h.handleGovernanceEffective)
	root.HandleFunc("/admin/v1/governance:drift-check", h.handleGovernanceDriftCheck)
	root.HandleFunc("/admin/v1/governance:policy", h.handleGovernancePolicyUpdate)
	root.HandleFunc("/admin/tenants/", h.handleTenantEvidence)
	root.HandleFunc("/admin/v1/cost:tenants", h.handleTenantCostReport)
	root.HandleFunc("/admin/v1/integrity:verify", h.handleIntegrityVerify)
	root.Handle("/", auth.HTTPAuthMiddleware(h.Verifier, mux))

	handler := h.instrumentHTTP(root)
	srv := &http.Server{
		Addr:              h.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if len(h.ReadyChecks) == 0 {
		h.Log.Warnf("no readiness checks configured; /readyz will report ready")
	}

	h.Log.Infof("HTTP listening on %s (proxying to %s)", h.Addr, h.GRPCAddr)
	return srv.ListenAndServe()
}

func (h *HTTPServer) instrumentHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UTC().UnixNano())
		}
		r.Header.Set("X-Request-Id", requestID)
		ww := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		ww.Header().Set("X-Request-Id", requestID)
		start := time.Now()
		next.ServeHTTP(ww, r)
		route := r.URL.Path
		if route == "" {
			route = "/"
		}
		observability.DefaultMetrics.ObserveHTTPRequest(r.Method, route, ww.code, time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

type oidcAuthzError struct {
	Code          string `json:"code"`
	Reason        string `json:"reason"`
	TenantID      string `json:"tenant_id,omitempty"`
	RequestID     string `json:"request_id"`
	CorrelationID string `json:"correlation_id"`
}

func gatewayErrorHandler(_ context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	st := status.Convert(err)
	if st.Code() != codes.PermissionDenied {
		runtime.DefaultHTTPErrorHandler(context.Background(), mux, marshaler, w, r, err)
		return
	}

	reason := "access_denied"
	tenantID := ""
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		if strings.TrimSpace(info.Reason) != "" {
			reason = strings.ToLower(strings.TrimSpace(info.Reason))
		}
		tenantID = strings.TrimSpace(info.Metadata["tenant_id"])
		break
	}

	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = fmt.Sprintf("req-%d", time.Now().UTC().UnixNano())
	}

	w.Header().Set("Content-Type", marshaler.ContentType(nil))
	w.WriteHeader(http.StatusForbidden)
	_ = marshaler.NewEncoder(w).Encode(map[string]any{
		"error": oidcAuthzError{
			Code:          "AUTHZ_DENY",
			Reason:        reason,
			TenantID:      tenantID,
			RequestID:     requestID,
			CorrelationID: requestID,
		},
	})
}

func (h *HTTPServer) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if len(h.ReadyChecks) == 0 {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ready": true, "checks": []readinessCheckResult{}})
		return
	}
	results := make([]readinessCheckResult, 0, len(h.ReadyChecks))
	allReady := true
	for _, c := range h.ReadyChecks {
		cctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
		err := c.fn(cctx)
		cancel()
		if err != nil {
			allReady = false
			results = append(results, readinessCheckResult{Name: c.name, Ready: false, Reason: err.Error()})
			continue
		}
		results = append(results, readinessCheckResult{Name: c.name, Ready: true})
	}
	if !allReady {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ready": false, "checks": results})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"ready": true, "checks": results})
}
