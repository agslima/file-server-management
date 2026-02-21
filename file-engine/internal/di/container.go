package di

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	"github.com/example/file-engine/internal/app/tasks"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/config"
	"github.com/example/file-engine/internal/handlers"
	"github.com/example/file-engine/internal/identity"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/server"
	"github.com/example/file-engine/internal/services"
	storagefactory "github.com/example/file-engine/internal/storage/factory"
)

type Container struct {
	Config *config.Config
	Logger *logger.Logger
}

type Servers struct {
	GRPC *server.GRPCServer
	HTTP *server.HTTPServer
}

func BuildContainer(cfg *config.Config, logg *logger.Logger) *Container {
	return &Container{Config: cfg, Logger: logg}
}

func (c *Container) Servers() *Servers {
	rdb := redis.NewClient(&redis.Options{Addr: c.Config.RedisAddr})
	q := redisq.NewRedisQueue(rdb)
	var pgPool *pgxpool.Pool

	// Storage backend (same as worker)
	st, err := storagefactory.NewFromConfig(context.Background(), storagefactory.Config{
		Backend:           c.Config.StorageBackend,
		LocalBase:         c.Config.FileBaseRoot,
		S3Bucket:          c.Config.S3Bucket,
		S3Region:          c.Config.S3Region,
		S3Prefix:          c.Config.S3Prefix,
		S3Endpoint:        c.Config.S3Endpoint,
		S3AccessKeyID:     getenv("AWS_ACCESS_KEY_ID"),
		S3SecretAccessKey: getenv("AWS_SECRET_ACCESS_KEY"),
		S3SessionToken:    getenv("AWS_SESSION_TOKEN"),
		GCSBucket:         c.Config.GCSBucket,
		GCSPrefix:         c.Config.GCSPrefix,
	})
	if err != nil {
		c.Logger.Fatalf("storage init: %v", err)
	}

	var aclStore auth.ACLStore
	if c.Config.PostgresDSN != "" {
		pool, err := pgxpool.New(context.Background(), c.Config.PostgresDSN)
		if err != nil {
			c.Logger.Fatalf("pg pool: %v", err)
		}
		pgPool = pool
		aclStore = auth.NewPostgresACLStore(pool)
	} else {
		aclStore = auth.NewInMemoryACLStore()
	}

	verifier, err := auth.NewJWTVerifierWithOIDC(c.Config.JWTSecret, c.Config.JWTPublicKeyPEM, c.Config.JWTIssuer, c.Config.JWTAudience, c.Config.JWTJWKSURL, c.Config.JWTActorIDClaim)
	if err != nil {
		c.Logger.Fatalf("jwt verifier: %v", err)
	}

	tenantResolver := buildTenantResolver(pgPool)
	auditor := tasks.NewDualLayerAuditEmitter(c.Logger, pgPool, getenv("AUDIT_IMMUTABLE_SINK_PATH"))

	objSvc := services.NewObjectService(st)
	uploadSvc := services.NewUploadServiceWithLogger(st, adaptersecurity.BuildMalwareScannerFromEnv(), services.UploadPolicy{
		MaxObjectSizeBytes: envInt64("UPLOAD_MAX_OBJECT_SIZE_BYTES", 10*1024*1024),
		TenantQuotaBytes:   envInt64("UPLOAD_TENANT_QUOTA_BYTES", 100*1024*1024),
		TenantObjectLimit:  envInt64("UPLOAD_TENANT_OBJECT_LIMIT", 0),
		RequestTimeout:     time.Duration(envInt64("UPLOAD_REQUEST_TIMEOUT_MS", 30000)) * time.Millisecond,
		RequireCleanScan:   strings.EqualFold(getenv("UPLOAD_REQUIRE_CLEAN_SCAN"), "true"),
	}, c.Logger)
	if policyPath := strings.TrimSpace(getenv("GOVERNANCE_POLICY_FILE")); policyPath != "" {
		govPolicy, err := services.LoadGovernancePolicyFromFile(policyPath)
		if err != nil {
			c.Logger.Fatalf("governance policy: %v", err)
		}
		if err := uploadSvc.SetGovernancePolicy(govPolicy); err != nil {
			c.Logger.Fatalf("apply governance policy: %v", err)
		}
	}
	if sourcePath := strings.TrimSpace(getenv("GOVERNANCE_POLICY_SOURCE")); sourcePath != "" {
		env, err := services.LoadGovernancePolicyFromSource(sourcePath, strings.TrimSpace(getenv("GOVERNANCE_POLICY_SOURCE_HMAC_KEY")))
		if err != nil {
			c.Logger.Fatalf("governance policy source: %v", err)
		}
		uploadSvc.SetGovernanceSource(env.Policy, env.Version)
		interval := time.Duration(envInt64("GOVERNANCE_DRIFT_CHECK_INTERVAL_SECONDS", 60)) * time.Second
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				if uploadSvc.CheckGovernanceDrift("system") {
					c.Logger.Warnf("governance drift detected against source version=%s", env.Version)
				}
			}
		}()
	}
	grpcHandler := handlers.NewGRPCHandler(q, objSvc, uploadSvc, aclStore, tenantResolver, c.Logger, auditor)

	grpcSrv := server.NewGRPCServer(c.Config.GRPCAddr, c.Logger, verifier, aclStore, grpcHandler)
	httpSrv := server.NewHTTPServer(c.Config.HTTPAddr, c.Config.GRPCAddr, c.Logger, verifier, st, aclStore, uploadSvc, tenantResolver)
	httpSrv.Identity = identity.NewStore(pgPool)
	httpSrv.UploadAuditor = auditor
	httpSrv.AddReadyCheck("storage", func(ctx context.Context) error {
		_, err := st.List(ctx, "/")
		return err
	})
	httpSrv.AddReadyCheck("queue", func(ctx context.Context) error {
		return rdb.Ping(ctx).Err()
	})
	if pgPool != nil {
		httpSrv.AddReadyCheck("postgres", func(ctx context.Context) error {
			if err := pgPool.Ping(ctx); err != nil {
				return err
			}
			return nil
		})
	}

	return &Servers{GRPC: grpcSrv, HTTP: httpSrv}
}

func buildTenantResolver(pool *pgxpool.Pool) auth.TenantResolver {
	if pool != nil {
		return auth.NewPostgresTenantResolver(pool)
	}

	raw := strings.TrimSpace(getenv("TENANT_MEMBERSHIPS"))
	if raw == "" {
		return auth.NewDenyAllTenantResolver()
	}
	// format: "alice=acme,beta;bob=beta"
	seed := map[string][]string{}
	for entry := range strings.SplitSeq(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		user := strings.TrimSpace(parts[0])
		tenants := strings.Split(parts[1], ",")
		seed[user] = tenants
	}
	return auth.NewInMemoryTenantResolver(seed)
}

func getenv(k string) string {
	return os.Getenv(k)
}

func envInt64(k string, d int64) int64 {
	v := strings.TrimSpace(getenv(k))
	if v == "" {
		return d
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return d
	}
	return n
}
