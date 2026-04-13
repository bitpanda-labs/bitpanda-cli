package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// richMockServer creates a test HTTP server with richer data for parameter testing.
// It includes wallets with zero balances, multiple transaction types, and supports
// basic query-parameter filtering so that server-side filters can be exercised.
func richMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
			w.Write([]byte(`Unauthorized`))
			return
		}

		switch {
		// ── Wallets ──────────────────────────────────────────────────────
		case strings.HasPrefix(r.URL.Path, "/v1/wallets"):
			allWallets := []map[string]any{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.5", "last_credited_at": "2024-06-15T10:00:00Z"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "STAKING", "balance": "100.0", "last_credited_at": "2024-06-10T08:00:00Z"},
				{"wallet_id": "w3", "asset_id": "a3", "wallet_type": "", "balance": "0", "last_credited_at": ""},
				{"wallet_id": "w4", "asset_id": "a1", "wallet_type": "STAKING", "balance": "0.25", "last_credited_at": "2024-07-01T12:00:00Z"},
			}

			// Server-side asset_id filter
			if aid := r.URL.Query().Get("asset_id"); aid != "" {
				var filtered []map[string]any
				for _, wal := range allWallets {
					if wal["asset_id"] == aid {
						filtered = append(filtered, wal)
					}
				}
				allWallets = filtered
			}

			json.NewEncoder(w).Encode(map[string]any{
				"data":          allWallets,
				"has_next_page": false,
				"end_cursor":    "",
			})

		// ── Assets (list) ───────────────────────────────────────────────
		case r.URL.Path == "/v1/assets":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "a1", "name": "Bitcoin", "symbol": "BTC"},
					{"id": "a2", "name": "Ethereum", "symbol": "ETH"},
					{"id": "a3", "name": "Cardano", "symbol": "ADA"},
				},
				"has_next_page": false,
				"end_cursor":    "",
			})

		// ── Asset (single) ──────────────────────────────────────────────
		case strings.HasPrefix(r.URL.Path, "/v1/assets/"):
			assetID := strings.TrimPrefix(r.URL.Path, "/v1/assets/")
			assets := map[string]map[string]string{
				"a1": {"id": "a1", "name": "Bitcoin", "symbol": "BTC"},
				"a2": {"id": "a2", "name": "Ethereum", "symbol": "ETH"},
				"a3": {"id": "a3", "name": "Cardano", "symbol": "ADA"},
			}
			if a, ok := assets[assetID]; ok {
				json.NewEncoder(w).Encode(map[string]any{"data": a})
			} else {
				w.WriteHeader(404)
				w.Write([]byte(`Asset not found`))
			}

		// ── Ticker ──────────────────────────────────────────────────────
		case strings.HasPrefix(r.URL.Path, "/v1/ticker"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "95000.00", "price_change_day": "1.5"},
					{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3500.00", "price_change_day": "-0.8"},
					{"id": "a3", "name": "Cardano", "symbol": "ADA", "type": "cryptocoin", "currency": "EUR", "price": "0.45", "price_change_day": "3.2"},
					{"id": "a4", "name": "Apple", "symbol": "AAPL", "type": "stock", "currency": "EUR", "price": "180.50", "price_change_day": "0.3"},
					{"id": "a5", "name": "Gold", "symbol": "XAUG", "type": "metal", "currency": "EUR", "price": "2300.00", "price_change_day": "0.1"},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		// ── Transactions ────────────────────────────────────────────────
		case strings.HasPrefix(r.URL.Path, "/v1/transactions"):
			allTxns := []map[string]any{
				{
					"transaction_id": "tx1", "asset_id": "a1", "wallet_id": "w1",
					"asset_amount": "0.5", "fee_amount": "0.001",
					"operation_type": "buy", "flow": "incoming",
					"credited_at": "2024-06-01T12:00:00Z", "trade_id": "trade1",
				},
				{
					"transaction_id": "tx2", "asset_id": "a2", "wallet_id": "w2",
					"asset_amount": "10.0", "fee_amount": "0.01",
					"operation_type": "buy", "flow": "incoming",
					"credited_at": "2024-06-05T14:00:00Z", "trade_id": "trade2",
				},
				{
					"transaction_id": "tx3", "asset_id": "a1", "wallet_id": "w1",
					"asset_amount": "0.1", "fee_amount": "0.0005",
					"operation_type": "sell", "flow": "incoming",
					"credited_at": "2024-07-10T09:00:00Z", "trade_id": "trade3",
				},
				{
					"transaction_id": "tx4", "asset_id": "a1", "wallet_id": "w1",
					"asset_amount": "0.2", "fee_amount": "0",
					"operation_type": "deposit", "flow": "incoming",
					"credited_at": "2024-07-15T16:00:00Z", "trade_id": "",
				},
				{
					"transaction_id": "tx5", "asset_id": "a2", "wallet_id": "w2",
					"asset_amount": "5.0", "fee_amount": "0.005",
					"operation_type": "transfer", "flow": "outgoing",
					"credited_at": "2024-08-01T10:00:00Z", "trade_id": "",
				},
			}

			q := r.URL.Query()

			// Server-side filters
			if wid := q.Get("wallet_id"); wid != "" {
				var filtered []map[string]any
				for _, tx := range allTxns {
					if tx["wallet_id"] == wid {
						filtered = append(filtered, tx)
					}
				}
				allTxns = filtered
			}
			if fl := q.Get("flow"); fl != "" {
				var filtered []map[string]any
				for _, tx := range allTxns {
					if tx["flow"] == fl {
						filtered = append(filtered, tx)
					}
				}
				allTxns = filtered
			}
			if aid := q.Get("asset_id"); aid != "" {
				var filtered []map[string]any
				for _, tx := range allTxns {
					if tx["asset_id"] == aid {
						filtered = append(filtered, tx)
					}
				}
				allTxns = filtered
			}

			json.NewEncoder(w).Encode(map[string]any{
				"data":          allTxns,
				"has_next_page": false,
				"end_cursor":    "",
			})

		default:
			w.WriteHeader(404)
			w.Write([]byte(`Not Found`))
		}
	}))
}

// ═══════════════════════════════════════════════════════════════════════════
// Global flag tests
// ═══════════════════════════════════════════════════════════════════════════

func TestGlobalAPIKeyFlag(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Use --api-key flag instead of env var
	stdout, _, code := runBP(t, []string{"BITPANDA_API_KEY=", "BITPANDA_BASE_URL=" + server.URL}, "balances", "--api-key", "test-key")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "1.5") {
		t.Errorf("expected balance 1.5 in output, got: %s", stdout)
	}
}

func TestOutputCSV(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// CSV output should contain comma-separated headers
	if !strings.Contains(stdout, "Wallet ID,") {
		t.Errorf("expected CSV header in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "1.5") {
		t.Errorf("expected balance in CSV output, got: %s", stdout)
	}
}

func TestOutputShortFlag(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Use -o shorthand
	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "-o", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "1.5") {
		t.Errorf("expected balance in JSON output, got: %s", stdout)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Balances command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestBalancesNonZero(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--non-zero")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// w3 has balance "0" — should be filtered out
	if strings.Contains(stdout, "w3") {
		t.Errorf("--non-zero should filter out zero-balance wallet w3, got: %s", stdout)
	}
	// w1 has balance 1.5 — should be present
	if !strings.Contains(stdout, "w1") {
		t.Errorf("expected wallet w1 in output, got: %s", stdout)
	}
}

func TestBalancesAssetIDFilter(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--asset-id", "a1")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should only show wallets with asset_id=a1 (w1 and w4)
	if !strings.Contains(stdout, "w1") {
		t.Errorf("expected wallet w1 in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "w4") {
		t.Errorf("expected wallet w4 in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "w2") {
		t.Errorf("wallet w2 (asset a2) should be filtered out, got: %s", stdout)
	}
}

func TestBalancesLimit(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--limit", "2")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Use JSON output for reliable row counting
	stdoutJSON, _, codeJSON := runBP(t, mockEnv(server.URL), "balances", "--limit", "2", "--output", "json")
	if codeJSON != 0 {
		t.Errorf("exit code = %d, want 0", codeJSON)
	}
	var rows []map[string]string
	if err := json.Unmarshal([]byte(stdoutJSON), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdoutJSON)
	}
	if len(rows) > 2 {
		t.Errorf("--limit 2 should produce at most 2 rows, got %d", len(rows))
	}
	_ = stdout // original table output verified the flag is accepted
}

func TestBalancesPageSize(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// page-size is an API parameter — just verify the flag is accepted and output works
	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--page-size", "10")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "w1") {
		t.Errorf("expected wallet data in output, got: %s", stdout)
	}
}

func TestBalancesCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least header + 1 data row, got %d lines", len(lines))
	}
	// First line should be CSV header
	if !strings.Contains(lines[0], "Wallet ID") {
		t.Errorf("expected CSV header, got: %s", lines[0])
	}
}

func TestBalancesCombinedFlags(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Combine --asset-id with --non-zero
	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--asset-id", "a1", "--non-zero")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "w1") {
		t.Errorf("expected wallet w1 (a1, balance 1.5), got: %s", stdout)
	}
	if !strings.Contains(stdout, "w4") {
		t.Errorf("expected wallet w4 (a1, balance 0.25), got: %s", stdout)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Portfolio command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestPortfolioSortByName(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "portfolio", "--sort", "name")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Bitcoin") {
		t.Errorf("expected Bitcoin in output, got: %s", stdout)
	}
	// Bitcoin should come before Ethereum alphabetically
	btcIdx := strings.Index(stdout, "Bitcoin")
	ethIdx := strings.Index(stdout, "Ethereum")
	if btcIdx > ethIdx {
		t.Errorf("expected Bitcoin before Ethereum with --sort name, got Bitcoin@%d, Ethereum@%d", btcIdx, ethIdx)
	}
}

func TestPortfolioSortByValue(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "portfolio", "--sort", "value")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "TOTAL") {
		t.Errorf("expected TOTAL row in output, got: %s", stdout)
	}
	// BTC value (1.75 * 95000 = 166250) > ETH value (100 * 3500 = 350000)
	// ETH should come first since 350000 > 166250
	btcIdx := strings.Index(stdout, "Bitcoin")
	ethIdx := strings.Index(stdout, "Ethereum")
	if ethIdx > btcIdx {
		t.Errorf("expected Ethereum (higher value) before Bitcoin with --sort value")
	}
}

func TestPortfolioJSONOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "portfolio", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// JSON should be valid
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, stdout)
	}
}

func TestPortfolioCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "portfolio", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Asset,") {
		t.Errorf("expected CSV headers, got: %s", stdout)
	}
	if !strings.Contains(stdout, "TOTAL") {
		t.Errorf("expected TOTAL row in CSV, got: %s", stdout)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Trades command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestTradesOperationBuy(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--operation", "buy")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "buy") {
		t.Errorf("expected buy trades, got: %s", stdout)
	}
	// sell trades should be excluded
	if strings.Contains(stdout, "sell") {
		t.Errorf("--operation buy should exclude sell trades, got: %s", stdout)
	}
}

func TestTradesOperationSell(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--operation", "sell")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "sell") {
		t.Errorf("expected sell trades, got: %s", stdout)
	}
}

func TestTradesAssetType(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--asset-type", "cryptocoin")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// BTC and ETH are cryptocoin, should be present
	if !strings.Contains(stdout, "Bitcoin") && !strings.Contains(stdout, "BTC") {
		t.Errorf("expected cryptocoin trades in output, got: %s", stdout)
	}
}

func TestTradesDateRange(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// --from and --to are passed to the API — just verify the flags are accepted
	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--from", "2024-06-01T00:00:00Z", "--to", "2024-06-30T00:00:00Z")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should get at least some results (mock doesn't actually date-filter)
	if stdout == "" {
		t.Error("expected some output with date range filter")
	}
}

func TestTradesLimit(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--limit", "1")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should have limited output — count "trade" occurrences in Trade ID column
	tradeCount := strings.Count(stdout, "trade")
	// At most 1 trade row (+ header mentions)
	if tradeCount > 3 {
		t.Errorf("--limit 1 should show at most 1 trade, found %d 'trade' occurrences:\n%s", tradeCount, stdout)
	}
}

func TestTradesJSONOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, stdout)
	}
}

func TestTradesCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Date,") {
		t.Errorf("expected CSV headers in trades output, got: %s", stdout)
	}
}

func TestTradesCombinedFlags(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Combine multiple flags
	stdout, _, code := runBP(t, mockEnv(server.URL), "trades",
		"--operation", "buy",
		"--asset-type", "cryptocoin",
		"--limit", "5",
		"--output", "json",
	)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON, got error: %v\noutput: %s", err, stdout)
	}
	for _, row := range result {
		if row["Operation"] != "buy" {
			t.Errorf("expected only buy operations, got: %s", row["Operation"])
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Transactions command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestTransactionsWalletID(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--wallet-id", "w1")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should only contain w1 transactions (tx1, tx3, tx4)
	if !strings.Contains(stdout, "tx1") {
		t.Errorf("expected tx1 in filtered output, got: %s", stdout)
	}
	if strings.Contains(stdout, "tx2") {
		t.Errorf("tx2 belongs to w2, should be filtered out, got: %s", stdout)
	}
	if strings.Contains(stdout, "tx5") {
		t.Errorf("tx5 belongs to w2, should be filtered out, got: %s", stdout)
	}
}

func TestTransactionsFlowIncoming(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--flow", "incoming")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "tx1") {
		t.Errorf("expected incoming tx1 in output, got: %s", stdout)
	}
	// tx5 is outgoing — should be filtered
	if strings.Contains(stdout, "tx5") {
		t.Errorf("tx5 is outgoing, should be filtered with --flow incoming, got: %s", stdout)
	}
}

func TestTransactionsFlowOutgoing(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--flow", "outgoing")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "tx5") {
		t.Errorf("expected outgoing tx5 in output, got: %s", stdout)
	}
	// tx1 is incoming — should be filtered
	if strings.Contains(stdout, "tx1") {
		t.Errorf("tx1 is incoming, should be filtered with --flow outgoing, got: %s", stdout)
	}
}

func TestTransactionsAssetID(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--asset-id", "a2")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// a2 transactions: tx2, tx5
	if !strings.Contains(stdout, "tx2") {
		t.Errorf("expected tx2 (a2) in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "tx1") {
		t.Errorf("tx1 (a1) should be filtered out, got: %s", stdout)
	}
}

func TestTransactionsDateRange(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions",
		"--from", "2024-06-01T00:00:00Z",
		"--to", "2024-08-01T00:00:00Z",
	)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout == "" {
		t.Error("expected some output with date range")
	}
}

func TestTransactionsLimit(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--limit", "2")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Verify limited output — count transaction IDs
	txCount := 0
	for _, id := range []string{"tx1", "tx2", "tx3", "tx4", "tx5"} {
		if strings.Contains(stdout, id) {
			txCount++
		}
	}
	if txCount > 2 {
		t.Errorf("--limit 2 should show at most 2 transactions, found %d:\n%s", txCount, stdout)
	}
}

func TestTransactionsPageSize(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--page-size", "50")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "tx1") {
		t.Errorf("expected tx1 in output, got: %s", stdout)
	}
}

func TestTransactionsJSONOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON, got error: %v\noutput: %s", err, stdout)
	}
}

func TestTransactionsCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Transaction ID,") {
		t.Errorf("expected CSV headers, got: %s", stdout)
	}
}

func TestTransactionsCombinedFilters(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Combine wallet-id + flow
	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions",
		"--wallet-id", "w1",
		"--flow", "incoming",
		"--output", "json",
	)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON, got error: %v\noutput: %s", err, stdout)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Price command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestPriceJSONOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "price", "ETH", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON, got error: %v\noutput: %s", err, stdout)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 price entry, got %d", len(result))
	}
	if result[0]["Symbol"] != "ETH" {
		t.Errorf("expected ETH symbol, got: %s", result[0]["Symbol"])
	}
}

func TestPriceCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "price", "BTC", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Symbol,") {
		t.Errorf("expected CSV header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "95000.00") {
		t.Errorf("expected BTC price in CSV, got: %s", stdout)
	}
}

func TestPriceLowerCaseSymbol(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// CLI should uppercase the symbol
	stdout, _, code := runBP(t, mockEnv(server.URL), "price", "btc")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "BTC") {
		t.Errorf("expected BTC in output (case-insensitive input), got: %s", stdout)
	}
}

func TestPriceUnknownSymbol(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	_, stderr, code := runBP(t, mockEnv(server.URL), "price", "NOSUCHCOIN")
	if code == 0 {
		t.Error("expected non-zero exit code for unknown symbol")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("expected 'not found' error, got: %s", stderr)
	}
}

func TestPriceMissingArg(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	_, stderr, code := runBP(t, mockEnv(server.URL), "price")
	if code == 0 {
		t.Error("expected non-zero exit code for missing symbol argument")
	}
	if !strings.Contains(stderr, "accepts 1 arg") {
		t.Errorf("expected argument error, got: %s", stderr)
	}
}

func TestPriceChangePercentage(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "price", "BTC")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should contain the 24h change with %
	if !strings.Contains(stdout, "1.5%") {
		t.Errorf("expected 24h change percentage in output, got: %s", stdout)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Prices command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestPricesAll(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "prices", "--all")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// --all should include ticker entries not held in wallet
	if !strings.Contains(stdout, "AAPL") {
		t.Errorf("--all should include AAPL (stock ticker), got: %s", stdout)
	}
	if !strings.Contains(stdout, "XAUG") {
		t.Errorf("--all should include XAUG (metal ticker), got: %s", stdout)
	}
}

func TestPricesHeldOnly(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Without --all, should only show prices for held assets
	stdout, _, code := runBP(t, mockEnv(server.URL), "prices")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// BTC and ETH are held (non-zero balance wallets)
	if !strings.Contains(stdout, "BTC") {
		t.Errorf("expected BTC (held asset) in prices, got: %s", stdout)
	}
	if !strings.Contains(stdout, "ETH") {
		t.Errorf("expected ETH (held asset) in prices, got: %s", stdout)
	}
}

func TestPricesJSONOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "prices", "--all", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON, got error: %v\noutput: %s", err, stdout)
	}
	if len(result) < 3 {
		t.Errorf("expected at least 3 ticker entries with --all, got %d", len(result))
	}
}

func TestPricesCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "prices", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Symbol,") {
		t.Errorf("expected CSV header, got: %s", stdout)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Asset command — all parameters
// ═══════════════════════════════════════════════════════════════════════════

func TestAssetJSONOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "asset", "a2", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	var result []map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("expected valid JSON, got error: %v\noutput: %s", err, stdout)
	}
	if len(result) != 1 || result[0]["Name"] != "Ethereum" {
		t.Errorf("expected Ethereum asset, got: %v", result)
	}
}

func TestAssetCSVOutput(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "asset", "a1", "--output", "csv")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "ID,") {
		t.Errorf("expected CSV header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Bitcoin") {
		t.Errorf("expected Bitcoin in CSV, got: %s", stdout)
	}
}

func TestAssetNotFound(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	_, stderr, code := runBP(t, mockEnv(server.URL), "asset", "nonexistent-id")
	if code == 0 {
		t.Error("expected non-zero exit code for unknown asset ID")
	}
	if stderr == "" {
		t.Error("expected error message for unknown asset")
	}
}

func TestAssetMissingArg(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	_, stderr, code := runBP(t, mockEnv(server.URL), "asset")
	if code == 0 {
		t.Error("expected non-zero exit code for missing asset ID argument")
	}
	if !strings.Contains(stderr, "accepts 1 arg") {
		t.Errorf("expected argument error, got: %s", stderr)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Completion command
// ═══════════════════════════════════════════════════════════════════════════

func TestCompletionBash(t *testing.T) {
	stdout, _, code := runBP(t, nil, "completion", "bash")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "bash") && !strings.Contains(stdout, "complete") {
		t.Errorf("expected bash completion script, got: %s", stdout[:min(200, len(stdout))])
	}
}

func TestCompletionZsh(t *testing.T) {
	stdout, _, code := runBP(t, nil, "completion", "zsh")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout == "" {
		t.Error("expected zsh completion output")
	}
}

func TestCompletionFish(t *testing.T) {
	stdout, _, code := runBP(t, nil, "completion", "fish")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout == "" {
		t.Error("expected fish completion output")
	}
}

func TestCompletionPowershell(t *testing.T) {
	stdout, _, code := runBP(t, nil, "completion", "powershell")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout == "" {
		t.Error("expected powershell completion output")
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	_, stderr, code := runBP(t, nil, "completion", "nushell")
	if code == 0 {
		t.Error("expected non-zero exit code for invalid shell")
	}
	if stderr == "" {
		t.Error("expected error message for invalid shell")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Error handling & edge cases
// ═══════════════════════════════════════════════════════════════════════════

func TestUnauthorizedExitCode(t *testing.T) {
	server := richMockServer(t)
	defer server.Close()

	// Use wrong API key
	_, stderr, code := runBP(t, []string{
		"BITPANDA_API_KEY=wrong-key",
		"BITPANDA_BASE_URL=" + server.URL,
	}, "balances")
	if code != 2 {
		t.Errorf("expected exit code 2 for auth error, got %d", code)
	}
	if !strings.Contains(stderr, "401") {
		t.Errorf("expected 401 in error message, got: %s", stderr)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, stderr, code := runBP(t, nil, "notacommand")
	if code == 0 {
		t.Error("expected non-zero exit code for unknown command")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %s", stderr)
	}
}

func TestUnknownFlag(t *testing.T) {
	_, stderr, code := runBP(t, []string{"BITPANDA_API_KEY=fake"}, "balances", "--nonexistent-flag")
	if code == 0 {
		t.Error("expected non-zero exit code for unknown flag")
	}
	if !strings.Contains(stderr, "unknown flag") {
		t.Errorf("expected 'unknown flag' error, got: %s", stderr)
	}
}

func TestEmptyResults(t *testing.T) {
	// Server that returns empty data for everything
	emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":          []any{},
			"has_next_page": false,
			"end_cursor":    "",
			"next_cursor":   "",
		})
	}))
	defer emptyServer.Close()

	t.Run("empty_balances", func(t *testing.T) {
		_, _, code := runBP(t, mockEnv(emptyServer.URL), "balances")
		if code != 0 {
			t.Errorf("exit code = %d, want 0 for empty balances", code)
		}
	})

	t.Run("empty_transactions", func(t *testing.T) {
		_, _, code := runBP(t, mockEnv(emptyServer.URL), "transactions")
		if code != 0 {
			t.Errorf("exit code = %d, want 0 for empty transactions", code)
		}
	})

	t.Run("empty_trades", func(t *testing.T) {
		_, _, code := runBP(t, mockEnv(emptyServer.URL), "trades")
		if code != 0 {
			t.Errorf("exit code = %d, want 0 for empty trades", code)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
