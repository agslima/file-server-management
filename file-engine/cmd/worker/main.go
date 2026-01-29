package main

import (
    "context"
    "os"
    "os/signal"
    "time"

    "github.com/redis/go-redis/v9"

    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/app/tasks"
    "github.com/example/file-engine/internal/config"
    "github.com/example/file-engine/internal/logger"
    "github.com/example/file-engine/internal/storage"
    "github.com/example/file-engine/internal/adapters/fs/local"
    "github.com/example/file-engine/internal/app/tasks"
)

func main() {
    cfg := config.LoadFromEnv()
    logg := logger.New(cfg.LogLevel)

    rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
    q := redisq.NewRedisQueue(rdb)

    // Choose backend
    st, err := storage.NewFromConfig(context.Background(), storage.FactoryConfig{
        Backend:   cfg.StorageBackend,
        LocalBase: cfg.FileBaseRoot,

        S3Bucket: cfg.S3Bucket,
        S3Region: cfg.S3Region,
        S3Prefix: cfg.S3Prefix,
        S3Endpoint: cfg.S3Endpoint,
        S3AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"),
        S3SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
        S3SessionToken: os.Getenv("AWS_SESSION_TOKEN"),

        GCSBucket: cfg.GCSBucket,
        GCSPrefix: cfg.GCSPrefix,
    })
    if err != nil {
        logg.Fatalf("storage init: %v", err)
    }

    proc := tasks.NewProcessorWithStorage(st)
    worker := tasks.NewWorker(q, proc, logg)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go worker.Start(ctx)

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, os.Interrupt)
    <-sig
    logg.Info("worker stopping")
    cancel()
    time.Sleep(500 * time.Millisecond)
}
