package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRunPortfolio_AggregatesMultipleWalletsSameAsset(t *testing.T) {
	// Two wallets for the same asset (BTC) should be aggregated into one row.
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.5"},
				{"wallet_id": "w2", "asset_id": "a1", "wallet_type": "STAKING", "balance": "0.5"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "100000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// Expect 2 rows: BTC aggregated + TOTAL
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (1 asset + TOTAL), got %d: %v", len(rows), rows)
	}

	btcRow := rows[0]
	if btcRow["Symbol"] != "BTC" {
		t.Errorf("expected symbol BTC, got %s", btcRow["Symbol"])
	}
	if btcRow["Balance"] != "2.00" {
		t.Errorf("expected aggregated balance 2.00, got %s", btcRow["Balance"])
	}
	// 2.0 * 100000 = 200000
	if btcRow["EUR Value"] != "200000.00" {
		t.Errorf("expected EUR value 200000.00, got %s", btcRow["EUR Value"])
	}

	totalRow := rows[1]
	if totalRow["Asset"] != "TOTAL" {
		t.Errorf("expected TOTAL row, got %s", totalRow["Asset"])
	}
	if totalRow["EUR Value"] != "200000.00" {
		t.Errorf("expected total 200000.00, got %s", totalRow["EUR Value"])
	}
}

func TestRunPortfolio_MultipleAssets(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "2.0"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "10.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
				{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// 2 assets + TOTAL = 3 rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Sorted by name: Bitcoin < Ethereum
	if rows[0]["Asset"] != "Bitcoin" {
		t.Errorf("expected first asset Bitcoin (sorted by name), got %s", rows[0]["Asset"])
	}
	if rows[1]["Asset"] != "Ethereum" {
		t.Errorf("expected second asset Ethereum, got %s", rows[1]["Asset"])
	}

	// Total = 2*50000 + 10*3000 = 100000 + 30000 = 130000
	if rows[2]["EUR Value"] != "130000.00" {
		t.Errorf("expected total 130000.00, got %s", rows[2]["EUR Value"])
	}
}

func TestRunPortfolio_SortByValue(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.0"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "100.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
				{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "value")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// ETH value = 100 * 3000 = 300000, BTC value = 1 * 50000 = 50000
	// Sorted by value descending: ETH first
	if rows[0]["Asset"] != "Ethereum" {
		t.Errorf("expected Ethereum first (highest value), got %s", rows[0]["Asset"])
	}
	if rows[1]["Asset"] != "Bitcoin" {
		t.Errorf("expected Bitcoin second, got %s", rows[1]["Asset"])
	}
}

func TestRunPortfolio_FiltersZeroBalances(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "0.0"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "5.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// Only ETH (non-zero) + TOTAL = 2 rows
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (1 asset + TOTAL), got %d", len(rows))
	}
	if rows[0]["Symbol"] != "ETH" {
		t.Errorf("expected ETH (zero-balance BTC filtered), got %s", rows[0]["Symbol"])
	}
}

func TestRunPortfolio_AllZeroBalances(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "0.0"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "0"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()
	var buf strings.Builder
	cmd.SetErr(&buf)

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	// Should produce no stdout (empty portfolio message goes to stderr)
	if raw != "" {
		t.Errorf("expected no stdout for all-zero balances, got: %s", raw)
	}
	if !strings.Contains(buf.String(), "No assets with balance found") {
		t.Errorf("expected 'No assets with balance found' on stderr, got: %s", buf.String())
	}
}

func TestRunPortfolio_EmptyWallets(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()
	var buf strings.Builder
	cmd.SetErr(&buf)

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if raw != "" {
		t.Errorf("expected no stdout for empty wallets, got: %s", raw)
	}
	if !strings.Contains(buf.String(), "No assets with balance found") {
		t.Errorf("expected empty portfolio message on stderr, got: %s", buf.String())
	}
}

func TestRunPortfolio_AssetWithNoTickerPrice(t *testing.T) {
	// Asset exists but has no ticker entry => price should be 0.
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "10.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			// Empty ticker
			w.Write(paginatedJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["EUR Price"] != "0.00" {
		t.Errorf("expected EUR price 0.00 for missing ticker, got %s", rows[0]["EUR Price"])
	}
	if rows[0]["EUR Value"] != "0.00" {
		t.Errorf("expected EUR value 0.00, got %s", rows[0]["EUR Value"])
	}
}

func TestRunPortfolio_InvalidBalanceSkipped(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "not-a-number"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "5.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()
	var stderrBuf strings.Builder
	cmd.SetErr(&stderrBuf)

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (ETH + TOTAL), got %d", len(rows))
	}
	if rows[0]["Symbol"] != "ETH" {
		t.Errorf("expected ETH, got %s", rows[0]["Symbol"])
	}

	// Check that a warning was emitted
	if !strings.Contains(stderrBuf.String(), "Warning: skipping wallet w1") {
		t.Errorf("expected warning about invalid balance, got: %s", stderrBuf.String())
	}
}

func TestRunPortfolio_WalletTypeAggregation(t *testing.T) {
	// Verify that different wallet types for the same asset are aggregated into one row.
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "3.0"},
				{"wallet_id": "w2", "asset_id": "a1", "wallet_type": "STAKING", "balance": "2.0"},
				{"wallet_id": "w3", "asset_id": "a1", "wallet_type": "SAVINGS", "balance": "1.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "10000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (BTC + TOTAL), got %d", len(rows))
	}
	// 3 + 2 + 1 = 6
	if rows[0]["Balance"] != "6.00" {
		t.Errorf("expected aggregated balance 6.00, got %s", rows[0]["Balance"])
	}
	// 6 * 10000 = 60000
	if rows[0]["EUR Value"] != "60000.00" {
		t.Errorf("expected EUR value 60000.00, got %s", rows[0]["EUR Value"])
	}
}

func TestRunPortfolio_UsesEURPriceWhenMultipleCurrencies(t *testing.T) {
	// Ticker returns the same asset with EUR, USD, and BTC prices.
	// Portfolio must use only the EUR price.
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "USD", "price": "99999.00"},
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "BTC", "price": "1.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (BTC + TOTAL), got %d", len(rows))
	}
	if rows[0]["EUR Price"] != "50000.00" {
		t.Errorf("expected EUR price 50000.00, got %s (wrong currency used)", rows[0]["EUR Price"])
	}
	if rows[0]["EUR Value"] != "50000.00" {
		t.Errorf("expected EUR value 50000.00, got %s", rows[0]["EUR Value"])
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "0.00"},
		{1.0, "1.00"},
		{1.006, "1.01"},
		{99999.999, "100000.00"},
		{0.123456, "0.12"},
	}

	for _, tt := range tests {
		got := formatFloat(tt.input)
		if got != tt.want {
			t.Errorf("formatFloat(%f) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestRunPortfolio_InvalidTickerPrice(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "5.0"},
			}))
		},
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			// Ticker entry with invalid price
			resp, _ := json.Marshal(map[string]interface{}{
				"data": []map[string]string{
					{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "bad-price"},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})
			w.Write(resp)
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()
	var stderrBuf strings.Builder
	cmd.SetErr(&stderrBuf)

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "name")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Price should fall back to 0.00
	if rows[0]["EUR Price"] != "0.00" {
		t.Errorf("expected EUR price 0.00 for invalid ticker price, got %s", rows[0]["EUR Price"])
	}
	if !strings.Contains(stderrBuf.String(), "Warning: invalid price") {
		t.Errorf("expected warning about invalid price, got: %s", stderrBuf.String())
	}
}
