package tasks

import (
	"context"
	"errors"
	"io"
	"strings"
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

func TestProcessorMissingParamErrorsAreExplicit(t *testing.T) {
	st := &processorTestStorage{}
	p := NewProcessorWithStorage(st)

	tests := []struct {
		name     string
		task     *redisq.TaskPayload
		expected string
	}{
		{
			name:     "create folder missing both",
			task:     &redisq.TaskPayload{Type: "create_folder", Params: map[string]string{}},
			expected: "missing params: parent,name",
		},
		{
			name:     "move file missing dst",
			task:     &redisq.TaskPayload{Type: "move_file", Params: map[string]string{"src": "a"}},
			expected: "missing param: dst",
		},
		{
			name:     "governed delete missing path",
			task:     &redisq.TaskPayload{Type: "governed_delete", Params: map[string]string{}},
			expected: "missing param: path",
		},
		{
			name:     "quarantine restore missing path",
			task:     &redisq.TaskPayload{Type: "quarantine_restore", Params: map[string]string{}},
			expected: "missing param: path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.Process(t.Context(), tt.task)
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
			if got := err.Error(); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestProcessorSeenKeysEvictToMaxEntries(t *testing.T) {
	st := &processorTestStorage{}
	p := NewProcessorWithStorage(st)
	p.seenKeyMaxEntries = 2

	now := time.Unix(2_000, 0)
	p.now = func() time.Time { return now }

	makeTask := func(key string) *redisq.TaskPayload {
		return &redisq.TaskPayload{Type: "create_folder", Params: map[string]string{
			"parent":          "tenants/acme",
			"name":            "docs",
			"idempotency_key": key,
		}}
	}

	if err := p.Process(t.Context(), makeTask("k1")); err != nil {
		t.Fatalf("k1 process failed: %v", err)
	}
	now = now.Add(1 * time.Second)
	if err := p.Process(t.Context(), makeTask("k2")); err != nil {
		t.Fatalf("k2 process failed: %v", err)
	}
	now = now.Add(1 * time.Second)
	if err := p.Process(t.Context(), makeTask("k3")); err != nil {
		t.Fatalf("k3 process failed: %v", err)
	}

	if len(p.seenKeys) != 2 {
		t.Fatalf("expected seen keys to be capped at 2, got %d", len(p.seenKeys))
	}
	if _, ok := p.seenKeys["k1"]; ok {
		t.Fatal("expected oldest key k1 to be evicted")
	}

	now = now.Add(1 * time.Second)
	if err := p.Process(t.Context(), makeTask("k2")); err != nil {
		t.Fatalf("expected duplicate k2 to be skipped, got %v", err)
	}
	if st.calls != 3 {
		t.Fatalf("expected exactly 3 storage calls, got %d", st.calls)
	}
}

func TestMissingParamsErrorFormatting(t *testing.T) {
	if got := missingParamsError("parent").Error(); got != "missing param: parent" {
		t.Fatalf("unexpected single param error: %q", got)
	}
	if got := missingParamsError("parent", "name").Error(); !strings.Contains(got, "parent,name") {
		t.Fatalf("unexpected multi param error: %q", got)
	}
}
