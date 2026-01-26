
package worker

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
)

type RedisQueue struct {
    Client *redis.Client
}

func (q *RedisQueue) Pop(ctx context.Context) (*Task, error) {
    res, err := q.Client.BLPop(ctx, 0, "tasks").Result()
    if err != nil {
        return nil, err
    }
    var t Task
    err = json.Unmarshal([]byte(res[1]), &t)
    return &t, err
}

func (q *RedisQueue) Complete(ctx context.Context, id string, status string) error {
    return q.Client.Set(ctx, "task:"+id, status, 0).Err()
}
