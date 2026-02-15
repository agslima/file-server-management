package redisq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskPayload struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Params map[string]string `json:"params"`
}

type TaskStatus struct {
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	LastTransition string `json:"last_transition"`
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
		if err == redis.Nil {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	var t TaskPayload
	if err := json.Unmarshal([]byte(res[1]), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (q *RedisQueue) Complete(ctx context.Context, id, status, correlationID, message string) error {
	return q.SetStatus(ctx, id, status, correlationID, message)
}

func (q *RedisQueue) SetStatus(ctx context.Context, id, status, correlationID, message string) error {
	payload := TaskStatus{
		TaskID:         id,
		Status:         status,
		Message:        message,
		CorrelationID:  correlationID,
		LastTransition: time.Now().UTC().Format(time.RFC3339Nano),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.client.Set(ctx, "task:"+id, string(b), 0).Err()
}

func (q *RedisQueue) GetStatus(ctx context.Context, id string) (*TaskStatus, error) {
	raw, err := q.client.Get(ctx, "task:"+id).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	var payload TaskStatus
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		// backward compatibility for pre-week3 plain string values
		return &TaskStatus{TaskID: id, Status: raw}, nil
	}
	if payload.TaskID == "" {
		payload.TaskID = id
	}
	return &payload, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, payload *TaskPayload) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.client.RPush(ctx, "tasks", string(b)).Err()
}

// Convenience helper used by the gRPC handler.
func (q *RedisQueue) EnqueueCreateFolder(ctx context.Context, parentPath, folderName, requestedBy, correlationID string) (string, error) {
	id := fmt.Sprintf("task-%s", newID())
	p := &TaskPayload{
		ID:   id,
		Type: "create_folder",
		Params: map[string]string{
			"parent":         parentPath,
			"name":           folderName,
			"by":             requestedBy,
			"correlation_id": correlationID,
		},
	}
	if err := q.Enqueue(ctx, p); err != nil {
		return "", err
	}
	if err := q.SetStatus(ctx, id, "queued", correlationID, "task accepted"); err != nil {
		return "", err
	}
	return id, nil
}

func newID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
