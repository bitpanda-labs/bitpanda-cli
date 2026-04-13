package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestRunPrices_AllFlag(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "-2.5"},
				{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00", "price_change_day": "1.2"},
				{"id": "a3", "name": "Gold", "symbol": "XAU", "type": "metal", "currency": "EUR", "price": "2000.00", "price_change_day": "0.3"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrices(cmd, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 3 {
		t.Fatalf("expected 3 prices with --all, got %d", len(rows))
	}
	// Should be sorted alphabetically
	if rows[0]["Symbol"] != "BTC" {
		t.Errorf("expected first symbol BTC (sorted), got %s", rows[0]["Symbol"])
	}
	if rows[1]["Symbol"] != "ETH" {
		t.Errorf("expected second symbol ETH, got %s", rows[1]["Symbol"])
	}
	if rows[2]["Symbol"] != "XAU" {
		t.Errorf("expected third symbol XAU, got %s", rows[2]["Symbol"])
	}
	// Verify columns
	if rows[0]["Price"] != "50000.00" {
		t.Errorf("expected price 50000.00, got %s", rows[0]["Price"])
	}
	if rows[0]["Currency"] != "EUR" {
		t.Errorf("expected currency EUR, got %s", rows[0]["Currency"])
	}
	if rows[0]["24h Change"] != "-2.5%" {
		t.Errorf("expected 24h change -2.5%%, got %s", rows[0]["24h Change"])
	}
}

func TestRunPrices_HeldAssetsOnly(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0.5"},
				{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00", "price_change_day": "1.2"},
				{"id": "a3", "name": "Gold", "symbol": "XAU", "type": "metal", "currency": "EUR", "price": "2000.00", "price_change_day": "0.3"},
			}))
		},
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.5"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "0.0"}, // zero balance, excluded
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrices(cmd, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// Only BTC held with non-zero balance
	if len(rows) != 1 {
		t.Fatalf("expected 1 price (held only), got %d", len(rows))
	}
	if rows[0]["Symbol"] != "BTC" {
		t.Errorf("expected BTC, got %s", rows[0]["Symbol"])
	}
}

func TestRunPrices_NoHeldAssets(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrices(cmd, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for no held assets, got %d", len(rows))
	}
}

func TestRunPrices_InvalidBalanceSkipped(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "not-a-number"},
				{"wallet_id": "w2", "asset_id": "a1", "wallet_type": "", "balance": "1.0"},
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
		runErr = app.runPrices(cmd, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (invalid balance skipped), got %d", len(rows))
	}
	if rows[0]["Symbol"] != "BTC" {
		t.Errorf("expected BTC, got %s", rows[0]["Symbol"])
	}
	if !strings.Contains(stderrBuf.String(), "Warning: skipping wallet w1") {
		t.Errorf("expected warning about invalid balance, got: %s", stderrBuf.String())
	}
}

func TestRunPrices_SymbolNotInTicker(t *testing.T) {
	// Held asset whose ID doesn't appear in ticker should be silently skipped.
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a99", "wallet_type": "", "balance": "5.0"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrices(cmd, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows (asset not in ticker), got %d", len(rows))
	}
}

func TestRunPrices_DeduplicatesHeldSymbols(t *testing.T) {
	// Two wallets for the same asset should produce only one price row.
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.0"},
				{"wallet_id": "w2", "asset_id": "a1", "wallet_type": "STAKING", "balance": "2.0"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrices(cmd, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (deduplicated), got %d", len(rows))
	}
}

func TestRunPrices_TickerAPIError(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte(`Internal Server Error`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runPrices(cmd, true)
	if err == nil {
		t.Fatal("expected error for ticker API failure")
	}
}

func TestRunPrices_WalletsAPIError(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			w.Write([]byte(`Unauthorized`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runPrices(cmd, false)
	if err == nil {
		t.Fatal("expected error for wallets API failure")
	}
}
