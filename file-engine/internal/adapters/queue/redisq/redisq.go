package redisq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
	"github.com/redis/go-redis/v9"
)

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrQueueUnavailable = errors.New("queue unavailable")
)

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
	client              *redis.Client
	log                 *logger.Logger
	queueAlertThreshold int64
}

func NewRedisQueue(client *redis.Client) *RedisQueue {
	threshold := int64(100)
	if raw := os.Getenv("ALERT_QUEUE_DEPTH_THRESHOLD"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			threshold = parsed
		}
	}
	return &RedisQueue{client: client, log: logger.New(os.Getenv("LOG_LEVEL")), queueAlertThreshold: threshold}
}

func (q *RedisQueue) Pop(ctx context.Context) (*TaskPayload, error) {
	res, err := q.client.BLPop(ctx, 0*time.Second, "tasks").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	var t TaskPayload
	if err := json.Unmarshal([]byte(res[1]), &t); err != nil {
		return nil, err
	}
	q.observeQueueDepth(ctx, t.ID, t.Params["correlation_id"])
	if enqRaw := t.Params["enqueued_at_unix_nano"]; enqRaw != "" {
		if enq, err := strconv.ParseInt(enqRaw, 10, 64); err == nil {
			observability.DefaultMetrics.ObserveQueueLagMs(time.Since(time.Unix(0, enq)).Milliseconds())
		}
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
	if err := q.client.Set(ctx, "task:"+id, string(b), 0).Err(); err != nil {
		return err
	}
	observability.DefaultMetrics.ObserveStatus(status)
	return nil
}

func (q *RedisQueue) GetStatus(ctx context.Context, id string) (*TaskStatus, error) {
	raw, err := q.client.Get(ctx, "task:"+id).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	var payload TaskStatus
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
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
	if err := q.client.RPush(ctx, "tasks", string(b)).Err(); err != nil {
		observability.DefaultMetrics.IncQueueBackpressureReject()
		q.log.Event("error", "queue.enqueue.rejected", map[string]any{"event": "queue.backpressure.reject", "reason": err.Error(), "task_id": payload.ID, "correlation_id": payload.Params["correlation_id"]})
		return fmt.Errorf("%w: %w", ErrQueueUnavailable, err)
	}
	observability.DefaultMetrics.IncEnqueued()
	q.observeQueueDepth(ctx, payload.ID, payload.Params["correlation_id"])
	return nil
}

func (q *RedisQueue) observeQueueDepth(ctx context.Context, taskID, correlationID string) {
	depth, err := q.client.LLen(ctx, "tasks").Result()
	if err != nil {
		return
	}
	observability.DefaultMetrics.SetQueueDepth(depth)
	if depth >= q.queueAlertThreshold {
		q.log.Event("warn", "queue depth alert threshold exceeded", map[string]any{
			"event":          "queue.depth.alert",
			"task_id":        taskID,
			"correlation_id": correlationID,
			"request_id":     correlationID,
			"queue_depth":    depth,
			"threshold":      q.queueAlertThreshold,
		})
	}
}

func (q *RedisQueue) EnqueueCreateFolder(ctx context.Context, parentPath, folderName, requestedBy, correlationID string) (string, error) {
	return q.enqueueMutation(ctx, "create_folder", map[string]string{
		"parent": parentPath,
		"name":   folderName,
		"by":     requestedBy,
	}, correlationID, nil)
}

func (q *RedisQueue) EnqueueObjectMove(ctx context.Context, sourcePath, destinationPath, actorID, tenantID, correlationID, idempotencyKey string) (string, error) {
	return q.enqueueMutation(ctx, "move_file", map[string]string{
		"src": sourcePath,
		"dst": destinationPath,
	}, correlationID, withActorTenantKey(actorID, tenantID, idempotencyKey))
}

func (q *RedisQueue) EnqueueGovernedDelete(ctx context.Context, objectPath, actorID, tenantID, correlationID, idempotencyKey string) (string, error) {
	return q.enqueueMutation(ctx, "governed_delete", map[string]string{
		"path": objectPath,
	}, correlationID, withActorTenantKey(actorID, tenantID, idempotencyKey))
}

func (q *RedisQueue) EnqueueQuarantineRestore(ctx context.Context, objectPath string, forceReprocess bool, actorID, tenantID, correlationID, idempotencyKey string) (string, error) {
	return q.enqueueMutation(ctx, "quarantine_restore", map[string]string{
		"path":            objectPath,
		"force_reprocess": strconv.FormatBool(forceReprocess),
	}, correlationID, withActorTenantKey(actorID, tenantID, idempotencyKey))
}

func withActorTenantKey(actorID, tenantID, key string) map[string]string {
	return map[string]string{
		"actor_id":        strings.TrimSpace(actorID),
		"tenant_id":       strings.TrimSpace(tenantID),
		"idempotency_key": strings.TrimSpace(key),
	}
}

func (q *RedisQueue) enqueueMutation(ctx context.Context, taskType string, params map[string]string, correlationID string, idempotency map[string]string) (string, error) {
	id := "task-" + newID()
	for k, v := range idempotency {
		if strings.TrimSpace(v) != "" {
			params[k] = strings.TrimSpace(v)
		}
	}
	params["correlation_id"] = correlationID
	params["request_id"] = correlationID
	params["enqueued_at_unix_nano"] = strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	if key := strings.TrimSpace(params["idempotency_key"]); key != "" {
		claimKey := "task:idempotency:" + key
		ok, err := q.client.SetNX(ctx, claimKey, id, 24*time.Hour).Result()
		if err != nil {
			return "", err
		}
		if !ok {
			existingID, err := q.client.Get(ctx, claimKey).Result()
			if err != nil {
				return "", err
			}
			return existingID, nil
		}
	}

	p := &TaskPayload{ID: id, Type: taskType, Params: params}
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
