package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
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
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
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
