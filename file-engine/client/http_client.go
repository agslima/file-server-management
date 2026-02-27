package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HTTPClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type APIError struct {
	StatusCode    int
	Code          string
	Reason        string
	Message       string
	RequestID     string
	CorrelationID string
	Retryable     bool
	RawBody       string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" || e.Reason != "" {
		return fmt.Sprintf("http %d %s/%s: %s", e.StatusCode, e.Code, e.Reason, e.Message)
	}
	return fmt.Sprintf("http %d: %s", e.StatusCode, strings.TrimSpace(e.RawBody))
}

func (e *APIError) Temporary() bool {
	if e == nil {
		return false
	}
	return e.Retryable || e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// AsAPIError attempts to extract an *APIError from err and reports whether extraction succeeded.
// If successful, it returns the extracted *APIError and true; otherwise it returns nil and false.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

type RetryOptions struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func (o RetryOptions) normalize() RetryOptions {
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 3
	}
	if o.BaseDelay <= 0 {
		o.BaseDelay = 200 * time.Millisecond
	}
	if o.MaxDelay <= 0 {
		o.MaxDelay = 2 * time.Second
	}
	return o
}

type UploadInitiateResponse struct {
	UploadID      string `json:"upload_id"`
	UploadURL     string `json:"upload_url"`
	StagingToken  string `json:"staging_token"`
	TargetPath    string `json:"target_path"`
	CorrelationID string `json:"correlation_id"`
}

type UploadCompleteResponse struct {
	UploadID      string `json:"upload_id"`
	Path          string `json:"path"`
	StagePath     string `json:"stage_path"`
	ScanStatus    string `json:"scan_status"`
	Checksum      string `json:"checksum"`
	CorrelationID string `json:"correlation_id"`
	Status        string `json:"status"`
}

type ReadinessResponse struct {
	Ready bool `json:"ready"`
}

// NewHTTPClient creates an HTTPClient configured with the provided baseURL and token.
// It trims any trailing '/' from baseURL, trims surrounding whitespace from token, and uses an http.Client with a 10-second timeout.
func NewHTTPClient(baseURL, token string) *HTTPClient {
	return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), token: strings.TrimSpace(token), client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *HTTPClient) doJSON(ctx context.Context, method, endpoint string, body, out any) error {
	var payload io.Reader = http.NoBody
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req) // #nosec G107 G704 -- Base URL is configured by trusted caller for this internal client.
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// parseAPIError creates an *APIError from an HTTP status and response body.
// If the response contains a top-level "error" JSON object with fields
// "code", "reason", "message", "request_id", "correlation_id", or "retryable",
// those values are copied into the returned APIError. The returned error always
// includes StatusCode and RawBody.
func parseAPIError(status int, raw []byte) error {
	apiErr := &APIError{StatusCode: status, RawBody: string(raw)}
	var payload struct {
		Error struct {
			Code          string `json:"code"`
			Reason        string `json:"reason"`
			Message       string `json:"message"`
			RequestID     string `json:"request_id"`
			CorrelationID string `json:"correlation_id"`
			Retryable     bool   `json:"retryable"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil && payload.Error.Code != "" {
		apiErr.Code = payload.Error.Code
		apiErr.Reason = payload.Error.Reason
		apiErr.Message = payload.Error.Message
		apiErr.RequestID = payload.Error.RequestID
		apiErr.CorrelationID = payload.Error.CorrelationID
		apiErr.Retryable = payload.Error.Retryable
	}
	return apiErr
}

func (c *HTTPClient) DoWithRetry(ctx context.Context, op func(context.Context) error, opt RetryOptions) error {
	opt = opt.normalize()
	var lastErr error
	for attempt := 1; attempt <= opt.MaxAttempts; attempt++ {
		if err := op(ctx); err != nil {
			lastErr = err
			if apiErr, ok := AsAPIError(err); ok && !apiErr.Temporary() {
				return err
			}
			if attempt == opt.MaxAttempts {
				break
			}
			delay := min(time.Duration(float64(opt.BaseDelay)*math.Pow(2, float64(attempt-1))), opt.MaxDelay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (c *HTTPClient) InitiateUpload(ctx context.Context, path string) (UploadInitiateResponse, error) {
	var out UploadInitiateResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/uploads:initiate", map[string]string{"path": path}, &out)
	return out, err
}

func (c *HTTPClient) UploadChunk(ctx context.Context, uploadID string, offset int64, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/v1/uploads/"+url.PathEscape(uploadID)+":chunk?offset="+strconv.FormatInt(offset, 10), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req) // #nosec G107 G704 -- Base URL is configured by trusted caller for this internal client.
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return parseAPIError(resp.StatusCode, raw)
	}
	return nil
}

func (c *HTTPClient) CompleteUpload(ctx context.Context, uploadID string) (UploadCompleteResponse, error) {
	var out UploadCompleteResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/uploads/"+url.PathEscape(uploadID)+":complete", nil, &out)
	return out, err
}

func (c *HTTPClient) Readyz(ctx context.Context) (ReadinessResponse, error) {
	var out ReadinessResponse
	err := c.doJSON(ctx, http.MethodGet, "/readyz", nil, &out)
	return out, err
}
