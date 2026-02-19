package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type readinessResponse struct {
	Ready  bool                  `json:"ready"`
	Checks []readinessCheckState `json:"checks"`
}

type readinessCheckState struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

func TestHandleReadyzReturnsReadyWhenChecksPass(t *testing.T) {
	h := &HTTPServer{}
	h.AddReadyCheck("db", func(context.Context) error { return nil })
	h.AddReadyCheck("queue", func(context.Context) error { return nil })
	h.AddReadyCheck("storage", func(context.Context) error { return nil })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	h.handleReadyz(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantChecks := []readinessCheckState{
		{Name: "db", Ready: true},
		{Name: "queue", Ready: true},
		{Name: "storage", Ready: true},
	}
	if !got.Ready {
		t.Fatalf("expected ready=true, got false: body=%s", rr.Body.String())
	}
	if !reflect.DeepEqual(got.Checks, wantChecks) {
		t.Fatalf("expected checks %+v, got %+v", wantChecks, got.Checks)
	}
}

func TestHandleReadyzReturnsServiceUnavailableWhenAnyCheckFails(t *testing.T) {
	h := &HTTPServer{}
	h.AddReadyCheck("db", func(context.Context) error { return nil })
	h.AddReadyCheck("queue", func(context.Context) error { return errors.New("redis unavailable") })
	h.AddReadyCheck("storage", func(context.Context) error { return nil })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	h.handleReadyz(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantChecks := []readinessCheckState{
		{Name: "db", Ready: true},
		{Name: "queue", Ready: false, Reason: "redis unavailable"},
		{Name: "storage", Ready: true},
	}
	if got.Ready {
		t.Fatalf("expected ready=false, got true: body=%s", rr.Body.String())
	}
	if !reflect.DeepEqual(got.Checks, wantChecks) {
		t.Fatalf("expected checks %+v, got %+v", wantChecks, got.Checks)
	}
}

func TestHandleReadyzWithoutChecksReturnsDeterministicReadyPayload(t *testing.T) {
	h := &HTTPServer{}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	h.handleReadyz(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.Ready {
		t.Fatalf("expected ready=true, got false")
	}
	if len(got.Checks) != 0 {
		t.Fatalf("expected empty checks, got %+v", got.Checks)
	}
}
