package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

// mockEndpoints maps URL path prefixes to handler functions.
type mockEndpoints map[string]http.HandlerFunc

// newMockServer creates a test server from a map of path prefix->handler.
// It matches the longest prefix first to avoid ambiguity.
func newMockServer(t *testing.T, endpoints mockEndpoints) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bestPrefix string
		var bestHandler http.HandlerFunc
		for prefix, handler := range endpoints {
			if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
				if len(prefix) > len(bestPrefix) {
					bestPrefix = prefix
					bestHandler = handler
				}
			}
		}
		if bestHandler != nil {
			bestHandler(w, r)
			return
		}
		w.WriteHeader(404)
		w.Write([]byte(`Not Found`))
	}))
}

// newTestApp creates an App with JSON output and a client pointing at the test server.
func newTestApp(serverURL string) *App {
	client := api.NewClient("test-key")
	client.BaseURL = serverURL
	return &App{
		apiClient: client,
		outFormat: output.FormatJSON,
	}
}

// paginatedJSON wraps data in the paginated response envelope.
func paginatedJSON(t *testing.T, data interface{}) []byte {
	t.Helper()
	dataBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshaling data: %v", err)
	}
	resp := map[string]interface{}{
		"data":          json.RawMessage(dataBytes),
		"has_next_page": false,
		"end_cursor":    "",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshaling response: %v", err)
	}
	return b
}

// assetJSON returns a JSON response for a single asset.
func assetJSON(id, name, symbol string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"data": map[string]string{"id": id, "name": name, "symbol": symbol},
	})
	return b
}

// assetsListHandler returns an http.HandlerFunc that serves a paginated list of assets.
func assetsListHandler(t *testing.T, assets ...[]string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		data := make([]map[string]string, len(assets))
		for i, a := range assets {
			data[i] = map[string]string{"id": a[0], "name": a[1], "symbol": a[2]}
		}
		w.Write(paginatedJSON(t, data))
	}
}

// captureStdout captures os.Stdout during fn execution and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()
	return string(buf[:n])
}

// parseJSONOutput parses captured JSON output into []map[string]string.
func parseJSONOutput(t *testing.T, raw string) []map[string]string {
	t.Helper()
	var result []map[string]string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parsing JSON output: %v\nraw: %s", err, raw)
	}
	return result
}

// newTestCmd creates a cobra.Command with a background context set.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

// nopWriter discards all writes.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
