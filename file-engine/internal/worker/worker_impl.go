package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/example/file-engine/internal/adapters/fs/local"
)

// Reuse Task struct from previous design
type Task struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Params map[string]string `json:"params"`
}

// RedisQueue implements Queue interface using redis list 'tasks'
type RedisQueue struct {
	Client *redis.Client
}

func NewRedisQueue(opt *redis.Options) *RedisQueue {
	return &RedisQueue{Client: redis.NewClient(opt)}
}

func (q *RedisQueue) Pop(ctx context.Context) (*Task, error) {
	res, err := q.Client.BLPop(ctx, 0*time.Second, "tasks").Result()
	if err != nil {
		return nil, err
	}
	var t Task
	if err := json.Unmarshal([]byte(res[1]), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (q *RedisQueue) Complete(ctx context.Context, id string, status string) error {
	return q.Client.Set(ctx, "task:"+id, status, 0).Err()
}

// FSProcessor now uses filesystem.LocalFs
type FSProcessor struct {
	FS *local.LocalFs
}

func NewFSProcessor(fs *local.LocalFs) *FSProcessor {
	return &FSProcessor{FS: fs}
}

func (p *FSProcessor) Process(ctx context.Context, task *Task) error {
	switch task.Type {
	case "create_folder":
		// expects param "path" and "folder"
		path := task.Params["path"]
		folder := task.Params["folder"]
		if path == "" || folder == "" {
			return fmt.Errorf("missing params")
		}
		// create folder under base: path/folder
		return p.FS.CreateFolder(ctx, path, folder)
	case "complete_upload":
		// expects upload_tmp and target_path and filename
		uploadTmp := task.Params["upload_tmp"]
		targetPath := task.Params["target_path"]
		filename := task.Params["filename"]
		if uploadTmp == "" || targetPath == "" || filename == "" {
			return fmt.Errorf("missing params for complete_upload")
		}
		// move file from tmp to final
		return p.FS.MoveUploadedFile(ctx, []string{uploadTmp}, []string{targetPath, filename})
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}
