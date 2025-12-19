package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/example/file-engine/internal/adapters/fs/local"
    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/app/tasks"
    "github.com/example/file-engine/internal/config"
    "github.com/example/file-engine/internal/logger"
)

func main() {
    cfg := config.LoadFromEnv()
    logg := logger.New(cfg.LogLevel)

    // Redis client
    rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})

    // filesystem
    lf, err := local.NewLocalFs(cfg.FileBaseRoot)
    if err != nil {
        logg.Fatalf("localfs init: %v", err)
    }

    q := redisq.NewRedisQueue(rdb)
    proc := tasks.NewProcessor(lf)
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