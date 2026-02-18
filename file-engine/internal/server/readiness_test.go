package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleReadyzReturnsReadyWhenChecksPass(t *testing.T) {
	h := &HTTPServer{}
	h.AddReadyCheck("storage", func(context.Context) error { return nil })
	h.AddReadyCheck("queue", func(context.Context) error { return nil })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	h.handleReadyz(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHandleReadyzReturnsServiceUnavailableWhenAnyCheckFails(t *testing.T) {
	h := &HTTPServer{}
	h.AddReadyCheck("storage", func(context.Context) error { return nil })
	h.AddReadyCheck("queue", func(context.Context) error { return errors.New("redis unavailable") })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	h.handleReadyz(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}
