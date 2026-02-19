package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/file-engine/internal/config"
	"github.com/example/file-engine/internal/di"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
)

func main() {
	cfg := config.LoadFromEnv()
	logg := logger.New(cfg.LogLevel)

	traceCfg := observability.ResolveTracingConfigFromEnv("file-engine-api")
	shutdownTracing, err := observability.InitTracing(context.Background(), traceCfg)
	if err != nil {
		logg.Fatalf("tracing init: %v", err)
	}
	defer func() {
		if shutdownErr := shutdownTracing(context.Background()); shutdownErr != nil {
			logg.Printf("tracing shutdown: %v", shutdownErr)
		}
	}()

	container := di.BuildContainer(cfg, logg)

	servers := container.Servers()

	// start servers
	go func() {
		if err := servers.HTTP.Start(); err != nil {
			logg.Fatalf("http server: %v", err)
		}
	}()

	go func() {
		if err := servers.GRPC.Start(); err != nil {
			logg.Fatalf("grpc server: %v", err)
		}
	}()

	// wait signal for graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	logg.Info("shutdown signal received")
	// TODO: implement graceful stop with contexts
	time.Sleep(500 * time.Millisecond)
	logg.Info("exiting")
}
