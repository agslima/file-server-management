package di

import (
    "context"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"

    "github.com/example/file-engine/internal/auth"
    "github.com/example/file-engine/internal/config"
    "github.com/example/file-engine/internal/logger"
    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/handlers"
    "github.com/example/file-engine/internal/server"
    "github.com/example/file-engine/internal/services"
    "github.com/example/file-engine/internal/storage"
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
    // Redis queue
    rdb := redis.NewClient(&redis.Options{Addr: c.Config.RedisAddr})
    q := redisq.NewRedisQueue(rdb)

    // Storage backend (same as worker)
    st, err := storage.NewFromConfig(context.Background(), storage.FactoryConfig{
        Backend:    c.Config.StorageBackend,
        LocalBase:  c.Config.FileBaseRoot,
        S3Bucket:   c.Config.S3Bucket,
        S3Region:   c.Config.S3Region,
        S3Prefix:   c.Config.S3Prefix,
        S3Endpoint: c.Config.S3Endpoint,
        S3AccessKeyID:     getenv("AWS_ACCESS_KEY_ID"),
        S3SecretAccessKey: getenv("AWS_SECRET_ACCESS_KEY"),
        S3SessionToken:    getenv("AWS_SESSION_TOKEN"),
        GCSBucket: c.Config.GCSBucket,
        GCSPrefix: c.Config.GCSPrefix,
    })
    if err != nil {
        c.Logger.Fatalf("storage init: %v", err)
    }

    // ACL store
    var aclStore auth.ACLStore
    if c.Config.PostgresDSN != "" {
        pool, err := pgxpool.New(context.Background(), c.Config.PostgresDSN)
        if err != nil {
            c.Logger.Fatalf("pg pool: %v", err)
        }
        aclStore = auth.NewPostgresACLStore(pool)
    } else {
        aclStore = auth.NewInMemoryACLStore()
    }

    // JWT verifier
    verifier, err := auth.NewJWTVerifier(c.Config.JWTSecret, c.Config.JWTPublicKeyPEM, c.Config.JWTIssuer, c.Config.JWTAudience)
    if err != nil {
        c.Logger.Fatalf("jwt verifier: %v", err)
    }

    // Services + Handlers
    objSvc := services.NewObjectService(st)
    grpcHandler := handlers.NewGRPCHandler(q, objSvc)

    grpcSrv := server.NewGRPCServer(c.Config.GRPCAddr, c.Logger, verifier, aclStore, grpcHandler)
    httpSrv := server.NewHTTPServer(c.Config.HTTPAddr, c.Config.GRPCAddr, c.Logger, verifier, st, aclStore)

    return &Servers{GRPC: grpcSrv, HTTP: httpSrv}
}

func getenv(k string) string {
    // avoid importing os everywhere
    // (no default here)
    return os.Getenv(k)
}
