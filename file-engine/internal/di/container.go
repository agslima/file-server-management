package di

import (
    "github.com/example/file-engine/internal/config"
    "github.com/example/file-engine/internal/logger"
    "github.com/example/file-engine/internal/adapters/fs/local"
    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/app/tasks"
    "github.com/example/file-engine/internal/server"
    "github.com/redis/go-redis/v9"
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
    lf, _ := local.NewLocalFs(c.Config.FileBaseRoot)
    q := redisq.NewRedisQueue(rdb)
    proc := tasks.NewProcessor(lf)
    _ = tasks.NewWorker(q, proc, c.Logger) // create worker but not started here

    grpcSrv := server.NewGRPCServer(c.Config.GRPCAddr)
    httpSrv := server.NewHTTPServer(c.Config.HTTPAddr, c.Config.GRPCAddr)
    return &Servers{GRPC: grpcSrv, HTTP: httpSrv}
}