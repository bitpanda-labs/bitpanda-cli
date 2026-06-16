// Package api provides an HTTP client for the Bitpanda Public API,
// including authentication, pagination, and typed endpoint methods.
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.public.bitpanda.com"

const (
	// defaultUserAgent is a version-agnostic fallback used only when a caller
	// never invokes SetUserAgent. The CLI always overrides this with the real
	// linker-injected Version (see internal/cli.Version), so it cannot go stale.
	// It carries no hardcoded release number; "dev" mirrors the in-code Version
	// default.
	defaultUserAgent = "bitpanda-cli/dev"
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
			// CheckRedirect follows redirects but strips the API key when a
			// redirect changes hosts, so the key is never leaked cross-host.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("stopped after 10 redirects")
				}
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					req.Header.Del("X-Api-Key")
				}
				return nil
			},
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
	return c.do(ctx, "GET", reqURL, nil)
}

// GetJSON performs a GET request and unmarshals the JSON response.
func (c *Client) GetJSON(ctx context.Context, path string, params url.Values, v any) error {
	body, err := c.Get(ctx, path, params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// PostJSON performs a POST request with a JSON-encoded body and unmarshals the
// JSON response. A nil body sends an empty request body.
func (c *Client) PostJSON(ctx context.Context, path string, body, v any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	respBody, err := c.do(ctx, "POST", c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	return json.Unmarshal(respBody, v)
}

// do executes an HTTP request with auth headers set and returns the response
// body, mapping non-2xx responses to an *APIError.
func (c *Client) do(ctx context.Context, method, reqURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    sanitizeBody(respBody),
		}
	}

	return respBody, nil
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// errorEnvelope matches the API's structured error response: {"error":{"code":"..."}}.
type errorEnvelope struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

// sanitizeBody extracts a human-readable message from an error response body.
// It prefers the structured {"error":{"code":...}} envelope, then falls back to
// stripping HTML and truncating raw text.
func sanitizeBody(b []byte) string {
	var env errorEnvelope
	if json.Unmarshal(b, &env) == nil && env.Error.Code != "" {
		return env.Error.Code
	}

	s := htmlTagRe.ReplaceAllString(string(b), "")
	s = stripControlChars(s)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// stripControlChars removes non-printable control characters (bytes < 0x20
// and DEL 0x7f) so a hostile endpoint cannot inject ANSI escape or other
// terminal control sequences. Common whitespace (space, tab, newline,
// carriage return) is preserved.
func stripControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}
