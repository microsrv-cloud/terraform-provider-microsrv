package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a thin Microsrv REST client.
type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

// Config configures the platform client.
type Config struct {
	Endpoint       string
	Token          string
	RequestTimeout time.Duration
}

// New creates a Microsrv API client. endpoint should be the platform base URL without trailing slash.
func New(cfg Config) (*Client, error) {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		endpoint: endpoint,
		token:    cfg.Token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// APIError is a non-2xx API response.
type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API %d: %s", e.StatusCode, e.Message)
	}
	if e.Body != "" {
		return fmt.Sprintf("API %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("API %d", e.StatusCode)
}

func (e *APIError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }
func (e *APIError) IsConflict() bool { return e.StatusCode == http.StatusConflict }
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

func (c *Client) do(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	u := c.endpoint + path
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp.StatusCode, raw)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// newAPIError builds an APIError from a non-2xx response body. A 401 gets a
// minting hint: The API now authenticates automation with access tokens
// ("sel_…", shown once at POST /api/v1/tokens); legacy session JWTs are dead.
func newAPIError(status int, raw []byte) *APIError {
	msg := strings.TrimSpace(string(raw))
	var errObj struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &errObj) == nil {
		if errObj.Error != "" {
			msg = errObj.Error
		} else if errObj.Message != "" {
			msg = errObj.Message
		}
	}
	if status == http.StatusUnauthorized {
		msg += "; create an access token (sel_…) at POST /api/v1/tokens or the console's Access → Tokens page and pass it as token/api_key"
	}
	return &APIError{StatusCode: status, Body: string(raw), Message: msg}
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// getText fetches a plain-text endpoint (e.g. log tails).
func (c *Client) getText(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/plain")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", newAPIError(resp.StatusCode, raw)
	}
	return string(raw), nil
}

// GetContainerLogs returns the tail of the container's stdout/stderr.
func (c *Client) GetContainerLogs(ctx context.Context, id string, tail int) (string, error) {
	return c.getText(ctx, withQuery("/api/v1/containers/"+id+"/logs", url.Values{"n": {fmt.Sprintf("%d", tail)}}))
}

// GetVMLogs returns the tail of the VM console/serial log.
func (c *Client) GetVMLogs(ctx context.Context, id string, tail int) (string, error) {
	return c.getText(ctx, withQuery("/api/v1/vms/"+id+"/logs", url.Values{"n": {fmt.Sprintf("%d", tail)}}))
}

func (c *Client) post(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPost, path, in, out)
}

func (c *Client) patch(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPatch, path, in, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// Query helpers

func withQuery(path string, q url.Values) string {
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}
