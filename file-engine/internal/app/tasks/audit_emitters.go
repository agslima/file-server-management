package tasks

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditEventRecord struct {
	Event         string            `json:"event"`
	TaskID        string            `json:"task_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Message       string            `json:"message,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type postgresAuditEmitter struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

func (e *postgresAuditEmitter) EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string) {
	if e.pool == nil {
		return
	}
	metadata := map[string]string{"source": "file-engine-worker"}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		e.log.Event("warn", "audit metadata marshal failed", map[string]any{"event": event, "task_id": taskID, "error": err.Error()})
		return
	}
	if _, err := e.pool.Exec(ctx,
		`INSERT INTO audit_events (event_type, task_id, correlation_id, message, metadata) VALUES ($1,$2,$3,$4,$5::jsonb)`,
		event, taskID, correlationID, message, string(metaJSON),
	); err != nil {
		e.log.Event("warn", "audit persistence failed", map[string]any{"event": event, "task_id": taskID, "error": err.Error()})
		observability.DefaultMetrics.IncAuditSinkFailure()
		return
	}
	observability.DefaultMetrics.IncAuditEventEmitted()
}

type sinkAuditEmitter struct {
	sink ImmutableSink
	log  *logger.Logger
}

func (e *sinkAuditEmitter) EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string) {
	rec := auditEventRecord{
		Event:         event,
		TaskID:        taskID,
		CorrelationID: correlationID,
		Message:       message,
		CreatedAt:     time.Now().UTC(),
		Metadata:      map[string]string{"source": "file-engine-worker", "sink": e.sink.Type()},
	}
	b, err := json.Marshal(rec)
	if err != nil {
		e.log.Event("warn", "immutable audit marshal failed", map[string]any{"event": event, "task_id": taskID, "error": err.Error()})
		return
	}
	if err := e.sink.WriteLine(ctx, b); err != nil {
		e.log.Event("warn", "immutable sink emission failed", map[string]any{"event": event, "task_id": taskID, "error": err.Error(), "sink": e.sink.Type()})
		observability.DefaultMetrics.IncAuditSinkFailure()
		return
	}
	observability.DefaultMetrics.IncAuditEventEmitted()
}

type multiAuditEmitter struct {
	emitters []AuditEmitter
}

func (m *multiAuditEmitter) EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string) {
	for _, e := range m.emitters {
		e.EmitTaskEvent(ctx, event, taskID, correlationID, message)
	}
}

func NewDualLayerAuditEmitter(logg *logger.Logger, pool *pgxpool.Pool, immutableSinkPath string) AuditEmitter {
	emitters := []AuditEmitter{&logAuditEmitter{log: logg}}
	if pool != nil {
		emitters = append(emitters, &postgresAuditEmitter{pool: pool, log: logg})
	}
	if sink := BuildImmutableSinkFromEnv(logg, strings.TrimSpace(immutableSinkPath)); sink != nil {
		emitters = append(emitters, &sinkAuditEmitter{sink: sink, log: logg})
	}
	if len(emitters) == 1 {
		return emitters[0]
	}
	return &multiAuditEmitter{emitters: emitters}
}
