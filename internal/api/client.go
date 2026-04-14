// Package api provides an HTTP client for the Bitpanda Developer API,
// including authentication, pagination, and typed endpoint methods.
package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const DefaultBaseURL = "https://developer.bitpanda.com"

const (
	defaultUserAgent = "bitpanda-cli/0.1.0"
	maxResponseSize  = 10 << 20 // 10 MB
)

// Client is the Bitpanda API HTTP client.
type Client struct {
	BaseURL    string
	APIKey     string
	UserAgent  string
	HTTPClient *http.Client
}

// NewClient creates a new API client.
func NewClient(apiKey string, insecure bool) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // user-requested via --insecure flag
	}
	return &Client{
		BaseURL:   DefaultBaseURL,
		APIKey:    apiKey,
		UserAgent: defaultUserAgent,
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// SetUserAgent overrides the default User-Agent header sent with every request.
func (c *Client) SetUserAgent(version string) {
	c.UserAgent = "bitpanda-cli/" + version
}

// APIError represents an error from the Bitpanda API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsAuthError returns true if this is a 401 authentication error.
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == 401
}

// Get performs a GET request to the given path with query parameters.
func (c *Client) Get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	reqURL := c.BaseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    sanitizeBody(string(body)),
		}
	}

	return body, nil
}

// GetJSON performs a GET request and unmarshals the JSON response.
func (c *Client) GetJSON(ctx context.Context, path string, params url.Values, v any) error {
	body, err := c.Get(ctx, path, params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// sanitizeBody truncates and strips HTML from error response bodies.
func sanitizeBody(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
