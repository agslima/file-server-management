package tasks

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/storage"
)

type processorTestStorage struct {
	createErr error
	calls     int
}

func (s *processorTestStorage) CreateFolder(context.Context, string) error {
	s.calls++
	if s.createErr != nil {
		return s.createErr
	}
	return nil
}

func (s *processorTestStorage) AtomicWrite(context.Context, string, io.Reader) error { return nil }
func (s *processorTestStorage) Move(context.Context, string, string) error           { return nil }
func (s *processorTestStorage) Delete(context.Context, string) error                 { return nil }
func (s *processorTestStorage) Exists(context.Context, string) (bool, error)         { return false, nil }
func (s *processorTestStorage) List(context.Context, string) ([]storage.ObjectInfo, error) {
	return nil, nil
}
func (s *processorTestStorage) Open(context.Context, string) (io.ReadCloser, error) { return nil, nil }

func TestProcessorIdempotencyMarksSeenOnlyAfterSuccess(t *testing.T) {
	st := &processorTestStorage{createErr: errors.New("boom")}
	p := NewProcessorWithStorage(st)
	task := &redisq.TaskPayload{Type: "create_folder", Params: map[string]string{
		"parent":          "tenants/acme",
		"name":            "docs",
		"idempotency_key": "idem-1",
	}}

	if err := p.Process(t.Context(), task); err == nil {
		t.Fatal("expected first run to fail")
	}

	st.createErr = nil
	if err := p.Process(t.Context(), task); err != nil {
		t.Fatalf("expected retry to execute and succeed, got %v", err)
	}
	if st.calls != 2 {
		t.Fatalf("expected task execution twice (failure then retry), got %d", st.calls)
	}
}

func TestProcessorIdempotencyEntriesExpire(t *testing.T) {
	st := &processorTestStorage{}
	p := NewProcessorWithStorage(st)

	now := time.Unix(1_000, 0)
	p.now = func() time.Time { return now }
	p.seenKeyTTL = 5 * time.Second

	task := &redisq.TaskPayload{Type: "create_folder", Params: map[string]string{
		"parent":          "tenants/acme",
		"name":            "docs",
		"idempotency_key": "idem-ttl",
	}}

	if err := p.Process(t.Context(), task); err != nil {
		t.Fatalf("expected first execution to succeed, got %v", err)
	}
	if err := p.Process(t.Context(), task); err != nil {
		t.Fatalf("expected duplicate execution to be skipped, got %v", err)
	}

	now = now.Add(6 * time.Second)
	if err := p.Process(t.Context(), task); err != nil {
		t.Fatalf("expected execution after TTL expiry to succeed, got %v", err)
	}

	if st.calls != 2 {
		t.Fatalf("expected executions before and after TTL expiry only, got %d", st.calls)
	}
}
