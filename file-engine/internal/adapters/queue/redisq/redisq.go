package redisq

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

type TaskPayload struct {
    ID     string            `json:"id"`
    Type   string            `json:"type"`
    Params map[string]string `json:"params"`
}

type RedisQueue struct {
    client *redis.Client
}

func NewRedisQueue(client *redis.Client) *RedisQueue {
    return &RedisQueue{client: client}
}

func (q *RedisQueue) Pop(ctx context.Context) (*TaskPayload, error) {
    res, err := q.client.BLPop(ctx, 0*time.Second, "tasks").Result()
    if err != nil {
        return nil, err
    }
    var t TaskPayload
    _ = json.Unmarshal([]byte(res[1]), &t)
    return &t, nil
}

func (q *RedisQueue) Complete(ctx context.Context, id, status string) error {
    return q.client.Set(ctx, "task:"+id, status, 0).Err()
}

func (q *RedisQueue) Enqueue(ctx context.Context, payload *TaskPayload) error {
    b, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    return q.client.RPush(ctx, "tasks", string(b)).Err()
}

// Convenience helper used by the gRPC handler.
func (q *RedisQueue) EnqueueCreateFolder(ctx context.Context, parentPath, folderName, requestedBy string) (string, error) {
    id := time.Now().UTC().Format("20060102T150405.000000000Z07:00")
    p := &TaskPayload{
        ID:   id,
        Type: "create_folder",
        Params: map[string]string{
            "parent": parentPath,
            "name":   folderName,
            "by":     requestedBy,
        },
    }
    if err := q.Enqueue(ctx, p); err != nil {
        return "", err
    }
    return id, nil
}
