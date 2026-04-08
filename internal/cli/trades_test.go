package cli

import (
	"net/http"
	"strconv"
	"testing"
)

// btcTickerHandler returns a ticker with a single BTC entry.
func btcTickerHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(paginatedJSON(t, []map[string]string{
			{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00"},
		}))
	}
}

func TestRunTrades_OperationFilter(t *testing.T) {
	tests := []struct {
		name      string
		txns      []map[string]interface{}
		operation string
		wantCount int
		wantOp    string // if set, all rows must have this operation
		wantID    string // if set, first row must have this trade ID
	}{
		{
			name: "buy filter returns only buys",
			txns: []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "t2", "flow": "incoming", "operation_type": "sell", "asset_amount": "0.5", "credited_at": "2024-01-02"},
				{"transaction_id": "tx3", "asset_id": "a1", "trade_id": "t3", "flow": "incoming", "operation_type": "buy", "asset_amount": "2.0", "credited_at": "2024-01-03"},
			},
			operation: "buy",
			wantCount: 2,
			wantOp:    "buy",
		},
		{
			name: "sell filter returns only sells",
			txns: []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "t2", "flow": "incoming", "operation_type": "sell", "asset_amount": "0.5", "credited_at": "2024-01-02"},
			},
			operation: "sell",
			wantCount: 1,
			wantOp:    "sell",
		},
		{
			name: "no filter excludes non-trade transactions",
			txns: []map[string]interface{}{
				// valid trade
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
				// no trade_id
				{"transaction_id": "tx2", "asset_id": "a1", "trade_id": "", "flow": "incoming", "operation_type": "deposit", "asset_amount": "2.0", "credited_at": "2024-01-02"},
				// outgoing flow
				{"transaction_id": "tx3", "asset_id": "a1", "trade_id": "t2", "flow": "outgoing", "operation_type": "buy", "asset_amount": "0.5", "credited_at": "2024-01-03"},
				// non buy/sell operation
				{"transaction_id": "tx4", "asset_id": "a1", "trade_id": "t3", "flow": "incoming", "operation_type": "transfer", "asset_amount": "0.1", "credited_at": "2024-01-04"},
			},
			operation: "",
			wantCount: 1,
			wantID:    "t1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := newMockServer(t, mockEndpoints{
				"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
					w.Write(paginatedJSON(t, tc.txns))
				},
				"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
				"/v1/ticker": btcTickerHandler(t),
			})
			defer server.Close()

			app := newTestApp(server.URL)
			cmd := newTestCmd()

			var runErr error
			raw := captureStdout(t, func() {
				runErr = app.runTrades(cmd, tc.operation, "", "", "", 0, true)
			})
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}

			rows := parseJSONOutput(t, raw)
			if len(rows) != tc.wantCount {
				t.Fatalf("expected %d rows, got %d", tc.wantCount, len(rows))
			}
			for _, row := range rows {
				if tc.wantOp != "" && row["Operation"] != tc.wantOp {
					t.Errorf("expected operation %q, got %q", tc.wantOp, row["Operation"])
				}
			}
			if tc.wantID != "" && len(rows) > 0 && rows[0]["Trade ID"] != tc.wantID {
				t.Errorf("expected trade ID %q, got %q", tc.wantID, rows[0]["Trade ID"])
			}
		})
	}
}

func TestRunTrades_AssetTypeFilter(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		wantCount int
		wantSym   string
	}{
		{"metal filter returns XAU", "metal", 1, "XAU"},
		{"stock filter returns nothing", "stock", 0, ""},
		{"cryptocoin filter returns BTC", "cryptocoin", 1, "BTC"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

			var runErr error
			raw := captureStdout(t, func() {
				runErr = app.runTrades(cmd, "", tc.assetType, "", "", 0, true)
			})
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}

			rows := parseJSONOutput(t, raw)
			if len(rows) != tc.wantCount {
				t.Fatalf("expected %d rows, got %d", tc.wantCount, len(rows))
			}
			if tc.wantSym != "" && len(rows) > 0 && rows[0]["Symbol"] != tc.wantSym {
				t.Errorf("expected symbol %q, got %q", tc.wantSym, rows[0]["Symbol"])
			}
		})
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
		"/v1/ticker": btcTickerHandler(t),
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 2, false)
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
	// limit=3 with --asset-type sets fetchLimit=3*10=30 to account for client-side filtering.
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			// 50 transactions: every 10th is metal, rest are crypto
			txns := make([]map[string]interface{}, 50)
			for i := range txns {
				assetID := "a1"
				if i%10 == 0 {
					assetID = "a2"
				}
				txns[i] = map[string]interface{}{
					"transaction_id": "tx" + strconv.Itoa(i),
					"asset_id":       assetID,
					"trade_id":       "trade" + strconv.Itoa(i),
					"flow":           "incoming",
					"operation_type": "buy",
					"asset_amount":   "1.0",
					"credited_at":    "2024-01-01",
				}
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

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "metal", "", "", 3, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows after limit, got %d", len(rows))
	}
	for _, row := range rows {
		if row["Type"] != "metal" {
			t.Errorf("expected type metal, got %s", row["Type"])
		}
	}
}

func TestRunTrades_AllFlagWithAssetType(t *testing.T) {
	// --all with --asset-type: fetchLimit stays 0 (no cap, no multiplication).
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": btcTickerHandler(t),
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "cryptocoin", "", "", 0, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestRunTrades_DefaultCapWithAssetType(t *testing.T) {
	// Without --all or --limit, default cap (100) is applied, then multiplied by 10
	// for asset-type filtering (fetchLimit=1000).
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{"transaction_id": "tx1", "asset_id": "a1", "trade_id": "t1", "flow": "incoming", "operation_type": "buy", "asset_amount": "1.0", "credited_at": "2024-01-01"},
			}))
		},
		"/v1/assets": assetsListHandler(t, []string{"a1", "Bitcoin", "BTC"}),
		"/v1/ticker": btcTickerHandler(t),
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "cryptocoin", "", "", 0, false)
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

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 0, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
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

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 0, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
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
		"/v1/assets": assetsListHandler(t),
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()
	cmd.SetErr(&nopWriter{})

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", "", 0, true)
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
