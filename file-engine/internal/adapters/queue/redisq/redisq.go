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