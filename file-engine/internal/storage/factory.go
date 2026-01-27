package storage

import (
    "context"
    "fmt"
    "strings"

    gs "github.com/example/file-engine/internal/adapters/storage/gcs"
    ls "github.com/example/file-engine/internal/adapters/storage/local"
    ss "github.com/example/file-engine/internal/adapters/storage/s3"
)

type FactoryConfig struct {
    Backend string // local|s3|gcs

    // local
    LocalBase string

    // s3
    S3Bucket string
    S3Region string
    S3Prefix string
    S3Endpoint string
    S3AccessKeyID string
    S3SecretAccessKey string
    S3SessionToken string

    // gcs
    GCSBucket string
    GCSPrefix string
}

func NewFromConfig(ctx context.Context, cfg FactoryConfig) (Storage, error) {
    b := strings.ToLower(strings.TrimSpace(cfg.Backend))
    if b == "" {
        b = "local"
    }
    switch b {
    case "local":
        return ls.New(cfg.LocalBase), nil
    case "s3":
        return ss.New(ctx, ss.Config{
            Bucket: cfg.S3Bucket, Region: cfg.S3Region, Prefix: cfg.S3Prefix,
            Endpoint: cfg.S3Endpoint, AccessKeyID: cfg.S3AccessKeyID,
            SecretAccessKey: cfg.S3SecretAccessKey, SessionToken: cfg.S3SessionToken,
        })
    case "gcs":
        return gs.New(ctx, gs.Config{Bucket: cfg.GCSBucket, Prefix: cfg.GCSPrefix})
    default:
        return nil, fmt.Errorf("unknown storage backend: %s", cfg.Backend)
    }
}
