package cli

import (
	"net/http"
	"strconv"
	"testing"

)

func TestRunTrades_FiltersBuyOnly(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "t2", "flow": "incoming", "operation_type": "sell", "asset_amount": "0.5", "credited_at": "2024-01-02"},
				{"transaction_id": "tx3", "asset_id": "a1", "trade_id": "t3", "flow": "incoming", "operation_type": "buy", "asset_amount": "2.0", "credited_at": "2024-01-03"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "buy", "", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 buy trades, got %d", len(rows))
	}
	for _, row := range rows {
		if row["Operation"] != "buy" {
			t.Errorf("expected operation 'buy', got %s", row["Operation"])
		}
	}
}

func TestRunTrades_FiltersSellOnly(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "t2", "flow": "incoming", "operation_type": "sell", "asset_amount": "0.5", "credited_at": "2024-01-02"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "sell", "", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 sell trade, got %d", len(rows))
	}
	if rows[0]["Operation"] != "sell" {
		t.Errorf("expected operation 'sell', got %s", rows[0]["Operation"])
	}
}

func TestRunTrades_ExcludesNonTradeTransactions(t *testing.T) {
	// Transactions without trade_id or not incoming should be excluded.
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				// Valid trade
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				// No trade_id (not a trade)
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "", "flow": "incoming", "operation_type": "deposit", "asset_amount": "2.0", "credited_at": "2024-01-02"},
				// Outgoing flow (not the trade incoming side)
				{"transaction_id": "tx3", "asset_id": "a1", "trade_id": "t2", "flow": "outgoing", "operation_type": "buy", "asset_amount": "0.5", "credited_at": "2024-01-03"},
				// Non buy/sell operation type (should be filtered when operation is empty)
				{"transaction_id": "tx4", "asset_id": "a1", "trade_id": "t3", "flow": "incoming", "operation_type": "transfer", "asset_amount": "0.1", "credited_at": "2024-01-04"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		// No operation filter => only buy/sell incoming trades
		runErr = app.runTrades(cmd, "", "", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 trade (only valid buy), got %d", len(rows))
	}
	if rows[0]["Trade ID"] != "t1" {
		t.Errorf("expected trade t1, got %s", rows[0]["Trade ID"])
	}
}

func TestRunTrades_AssetTypeFilter(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				{"transaction_id": "tx2", "asset_id": "a2", "trade_id": "t2", "flow": "incoming", "operation_type": "buy", "asset_amount": "10.0", "credited_at": "2024-01-02"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}, []string{"a2", "Gold", "XAU"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
				{"id": "t2", "symbol": "XAU", "type": "metal", "currency": "EUR", "price": "2000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "metal", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 metal trade, got %d", len(rows))
	}
	if rows[0]["Symbol"] != "XAU" {
		t.Errorf("expected XAU, got %s", rows[0]["Symbol"])
	}
	if rows[0]["Type"] != "metal" {
		t.Errorf("expected type metal, got %s", rows[0]["Type"])
	}
}

func TestRunTrades_AssetTypeFilterRemovesAll(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		// Filter by "stock" which doesn't match BTC (cryptocoin)
		runErr = app.runTrades(cmd, "", "stock", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows when filter removes all trades, got %d", len(rows))
	}
}

func TestRunTrades_LimitApplied(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "t2", "flow": "incoming", "operation_type": "buy", "asset_amount": "2.0", "credited_at": "2024-01-02"},
				{"transaction_id": "tx3", "asset_id": "a1", "trade_id": "t3", "flow": "incoming", "operation_type": "buy", "asset_amount": "3.0", "credited_at": "2024-01-03"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 2)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (limit=2), got %d", len(rows))
	}
}

func TestRunTrades_FetchLimitHeuristic(t *testing.T) {
	// When asset-type filter is set with a limit, the API should be called
	// with fetchLimit = limit * 10 to account for post-fetch filtering.
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			// Return 50 transactions: 5 are metal, rest are crypto
			txns := make([]map[string]interface{}, 0, 50)
			for i := 0; i < 50; i++ {
				assetID := "a1" // crypto
				if i%10 == 0 {
					assetID = "a2" // metal (5 of 50)
				}
				txns = append(txns, map[string]interface{}{
					"transaction_id": "tx" + strconv.Itoa(i),
					"asset_id":       assetID,
					"trade_id":       "trade" + strconv.Itoa(i),
					"flow":           "incoming",
					"operation_type": "buy",
					"asset_amount":   "1.0",
					"credited_at":    "2024-01-01",
				})
			}
			w.Write(paginatedJSON(t, txns))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}, []string{"a2", "Gold", "XAU"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
				{"id": "t2", "symbol": "XAU", "type": "metal", "currency": "EUR", "price": "2000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		// limit=3, asset-type=metal => fetchLimit should be 3*10=30
		runErr = app.runTrades(cmd, "", "metal", "", "", 3)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// We have 5 metal trades in 50 transactions, limited to 3
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows after limit, got %d", len(rows))
	}
	for _, row := range rows {
		if row["Type"] != "metal" {
			t.Errorf("expected type metal, got %s", row["Type"])
		}
	}
}

func TestRunTrades_NoLimitWithAssetType(t *testing.T) {
	// When limit=0, fetchLimit should remain 0 (no multiplication).
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "cryptocoin", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestRunTrades_EmptyTransactions(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{}))
		},
		"/v1/assets": assetsListHandler(t),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for empty transactions, got %d", len(rows))
	}
}

func TestRunTrades_EURPriceFormatted(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "95123.456789"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// Price should be formatted to 2 decimal places
	if rows[0]["EUR Price"] != "95123.46" {
		t.Errorf("expected formatted price 95123.46, got %s", rows[0]["EUR Price"])
	}
}

func TestRunTrades_UnknownAssetShowsUnknown(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "unknown-id", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
			}))
		},
		"/v1/assets": assetsListHandler(t), // empty list, so unknown-id won't resolve
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()
	cmd.SetErr(&nopWriter{})

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 0)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["Asset"] != "unknown" {
		t.Errorf("expected asset name 'unknown', got %s", rows[0]["Asset"])
	}
	if rows[0]["Symbol"] != "unknown" {
		t.Errorf("expected symbol 'unknown', got %s", rows[0]["Symbol"])
	}
	if rows[0]["EUR Price"] != "N/A" {
		t.Errorf("expected EUR price 'N/A', got %s", rows[0]["EUR Price"])
	}
}
