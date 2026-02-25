package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAsAPIErrorParsesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"THROTTLED","reason":"rate_limited","message":"rate limit exceeded","request_id":"req-1","correlation_id":"req-1","retryable":true}}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	_, err := c.InitiateUpload(context.Background(), "/tenants/acme/docs/a.txt")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError got %T", err)
	}
	if apiErr.Code != "THROTTLED" || !apiErr.Retryable {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestDoWithRetryRetriesTemporaryAPIError(t *testing.T) {
	attempt := 0
	c := NewHTTPClient("http://example", "")
	err := c.DoWithRetry(context.Background(), func(context.Context) error {
		attempt++
		if attempt < 3 {
			return &APIError{StatusCode: http.StatusTooManyRequests, Code: "THROTTLED", Retryable: true}
		}
		return nil
	}, RetryOptions{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond})
	if err != nil {
		t.Fatalf("expected nil err got %v", err)
	}
	if attempt != 3 {
		t.Fatalf("expected 3 attempts got %d", attempt)
	}
}

func TestDoWithRetryStopsOnPermanentError(t *testing.T) {
	attempt := 0
	c := NewHTTPClient("http://example", "")
	err := c.DoWithRetry(context.Background(), func(context.Context) error {
		attempt++
		return &APIError{StatusCode: http.StatusForbidden, Code: "UPLOAD_ERROR", Retryable: false}
	}, RetryOptions{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError got %T", err)
	}
	if attempt != 1 {
		t.Fatalf("expected 1 attempt got %d", attempt)
	}
}
