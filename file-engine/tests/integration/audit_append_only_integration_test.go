package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuditEventsAppendOnlyEnforced(t *testing.T) {
	t.Parallel()

	dsn := os.Getenv("FILEENGINE_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		dsn = "postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unavailable for append-only integration test: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres unavailable for append-only integration test: %v", err)
	}

	migrationPath := filepath.Join("..", "db", "migrations", "0003_create_audit_events.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration sql: %v", err)
	}

	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("apply audit_events migration: %v", err)
	}

	if _, err := pool.Exec(ctx, "TRUNCATE audit_events"); err != nil {
		t.Fatalf("truncate audit_events: %v", err)
	}

	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO audit_events (event_type, task_id, correlation_id, message, metadata) VALUES ($1,$2,$3,$4,$5::jsonb) RETURNING id`,
		"object.list", "task-1", "corr-append-only", "seed event", `{"tenant_id":"acme","actor_id":"alice"}`,
	).Scan(&id); err != nil {
		t.Fatalf("insert seed audit event: %v", err)
	}

	if _, err := pool.Exec(ctx, "UPDATE audit_events SET message = 'mutated' WHERE id = $1", id); err == nil {
		t.Fatal("expected UPDATE on audit_events to be rejected")
	} else if !isAppendOnlyErr(err) {
		t.Fatalf("expected append-only/permission error for UPDATE, got: %v", err)
	}

	if _, err := pool.Exec(ctx, "DELETE FROM audit_events WHERE id = $1", id); err == nil {
		t.Fatal("expected DELETE on audit_events to be rejected")
	} else if !isAppendOnlyErr(err) {
		t.Fatalf("expected append-only/permission error for DELETE, got: %v", err)
	}

	var rowCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_events WHERE id = $1", id).Scan(&rowCount); err != nil {
		t.Fatalf("count audit row: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected seed row to remain after rejected update/delete, got count=%d", rowCount)
	}
}

func isAppendOnlyErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		msg := strings.ToLower(pgErr.Message)
		if strings.Contains(msg, "append-only") || strings.Contains(msg, "permission denied") {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "append-only") || strings.Contains(msg, "permission denied")
}
