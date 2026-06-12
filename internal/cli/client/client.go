// Package client is a thin HTTP client for the GetPaidHQ API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// Do issues an API request and returns the raw response body. body may be
// nil, a json.RawMessage (sent verbatim — the --data path), or any value
// that marshals to JSON. Non-2xx responses become *APIError when the body
// carries the {code,message,details} envelope.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	if c.APIKey == "" && path != "/api/health" {
		return nil, errors.New("no API key configured: set GPHQ_API_KEY, add api_key to ~/.config/gphq/config.toml, or pass --api-key")
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("x-api-key", c.APIKey)
	}
	req.Header.Set("User-Agent", "gphq-cli")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return raw, nil
	}
	apiErr := &APIError{Status: resp.StatusCode}
	if jerr := json.Unmarshal(raw, apiErr); jerr == nil && (apiErr.Code != "" || apiErr.Message != "") {
		return nil, apiErr
	}
	return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
}

// ListQuery builds the standard pagination query parameters.
func ListQuery(page, limit int, sortBy, sortOrder string) url.Values {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("sort_by", sortBy)
	q.Set("sort_order", sortOrder)
	return q
}
