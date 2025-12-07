package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/example/file-engine/internal/worker"
	"github.com/example/file-engine/internal/filesystem"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisOpt := &redis.Options{Addr: redisAddr}

	// base root for filesystem operations (mounted volume)
	baseRoot := os.Getenv("FILE_BASE_ROOT")
	if baseRoot == "" {
		baseRoot = "/mnt/files"
	}

	// create LocalFs
	fs, err := filesystem.NewLocalFs(baseRoot)
	if err != nil {
		log.Fatalf("failed to create LocalFs: %v", err)
	}

	// create Redis queue and processor
	q := worker.NewRedisQueue(redisOpt)
	processor := worker.NewFSProcessor(fs)
	w := &worker.Worker{Queue: q, Processor: processor}

	// Run worker in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)

	// For demo: push a sample task if env PUSH_DEMO_TASK=true
	if os.Getenv("PUSH_DEMO_TASK") == "true" {
		t := worker.Task{ID: fmt.Sprintf("demo-%d", time.Now().Unix()), Type: "create_folder", Params: map[string]string{"path": "projects/demo", "folder": "created-by-worker"}}
		b, _ := json.Marshal(t)
		_ = q.Client.RPush(ctx, "tasks", string(b)).Err()
	}

	// wait shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Println("shutdown worker")
}