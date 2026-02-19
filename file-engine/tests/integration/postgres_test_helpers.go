// Package integration contains end-to-end integration tests for file-engine.
package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPostgresDSN() string {
	dsn := os.Getenv("FILEENGINE_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		dsn = "postgres://localhost:5432/fileengine?sslmode=disable"
	}
	return dsn
}

func mustConnectAuditDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, testPostgresDSN())
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping postgres: %v", err)
	}
	mustApplyAuditMigration(t, ctx, pool)
	return pool
}

func mustApplyAuditMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	migrationSQL, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "0003_create_audit_events.sql"))
	if err != nil {
		t.Fatalf("read migration sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("apply audit_events migration: %v", err)
	}
}

func mustResetAuditEvents(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, "TRUNCATE audit_events RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate audit_events: %v", err)
	}
}
