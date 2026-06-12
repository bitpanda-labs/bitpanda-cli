package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// paramMockServer records the most recent query parameters seen at /v1/operations
// and /v1/assets so tests can assert that CLI flags are forwarded correctly.
type paramMockServer struct {
	*httptest.Server
	lastOperationsQuery url.Values
	lastAssetsQuery     url.Values
}

func newParamMockServer(t *testing.T) *paramMockServer {
	t.Helper()
	pm := &paramMockServer{}
	pm.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
			w.Write([]byte(`Unauthorized`))
			return
		}

		switch {
		case r.URL.Path == "/v1/operations":
			pm.lastOperationsQuery = r.URL.Query()
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"operation_id":   "op1",
						"operation_type": "buy",
						"transactions": []map[string]any{
							{"transaction_id": "tx1", "asset_id": "a1", "flow": "INCOMING", "asset_amount": map[string]string{"value": "0.5"}, "credited_at": "2024-06-01T12:00:00Z", "trade": map[string]any{"trade_id": "trade1", "rate": "95000.00"}},
						},
					},
					{
						"operation_id":   "op2",
						"operation_type": "deposit",
						"transactions": []map[string]any{
							{"transaction_id": "tx2", "currency_id": "c1", "flow": "INCOMING", "asset_amount": map[string]string{"value": "1000.0"}, "credited_at": "2024-06-02T12:00:00Z"},
						},
					},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		case r.URL.Path == "/v1/assets":
			pm.lastAssetsQuery = r.URL.Query()
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin"},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		default:
			w.WriteHeader(404)
			w.Write([]byte(`Not Found`))
		}
	}))
	return pm
}

func TestOperationsForwardsDateAndAssetFilters(t *testing.T) {
	server := newParamMockServer(t)
	defer server.Close()

	_, _, code := runBP(t, mockEnv(server.URL), "operations",
		"--from", "2024-06-01T00:00:00Z",
		"--to", "2024-07-01T00:00:00Z",
		"--asset-id", "a1",
		"--asset-id", "a2",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	q := server.lastOperationsQuery
	if got := q.Get("from"); got != "2024-06-01T00:00:00Z" {
		t.Errorf("from = %q", got)
	}
	if got := q.Get("to"); got != "2024-07-01T00:00:00Z" {
		t.Errorf("to = %q", got)
	}
	if ids := q["asset_id"]; len(ids) != 2 {
		t.Errorf("asset_id = %v, want 2 values", ids)
	}
}

func TestOperationsPageSizeForwarded(t *testing.T) {
	server := newParamMockServer(t)
	defer server.Close()

	_, _, code := runBP(t, mockEnv(server.URL), "operations", "--page-size", "50")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := server.lastOperationsQuery.Get("page_size"); got != "50" {
		t.Errorf("page_size = %q, want 50", got)
	}
}

func TestTradesFiltersByAssetType(t *testing.T) {
	server := newParamMockServer(t)
	defer server.Close()

	// All trades have type cryptocoin in the mock; filtering by stock yields none.
	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--asset-type", "stock", "--output", "json")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if strings.Contains(stdout, "trade1") {
		t.Errorf("expected no stock trades, but trade1 was present: %s", stdout)
	}
	if !strings.Contains(stdout, "[]") {
		t.Errorf("expected empty JSON array for stock filter, got: %s", stdout)
	}

	// Filtering by cryptocoin keeps trade1.
	stdout, _, code = runBP(t, mockEnv(server.URL), "trades", "--asset-type", "cryptocoin")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "trade1") {
		t.Errorf("expected trade1 for cryptocoin filter, got: %s", stdout)
	}
}

func TestAssetsForwardsFilters(t *testing.T) {
	server := newParamMockServer(t)
	defer server.Close()

	_, _, code := runBP(t, mockEnv(server.URL), "assets", "--type", "cryptocoin", "--symbol", "BTC")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	q := server.lastAssetsQuery
	if got := q.Get("type"); got != "cryptocoin" {
		t.Errorf("type = %q, want cryptocoin", got)
	}
	if got := q.Get("symbol"); got != "BTC" {
		t.Errorf("symbol = %q, want BTC", got)
	}
}

func TestOperationsCSVOutput(t *testing.T) {
	server := newParamMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "operations", "--output", "csv")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Transaction ID") || !strings.Contains(stdout, "tx1") {
		t.Errorf("expected CSV header and rows, got: %s", stdout)
	}
}
