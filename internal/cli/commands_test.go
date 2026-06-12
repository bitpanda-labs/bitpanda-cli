package cli

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func TestRunPortfolio_EnrichesAndSorts(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/portfolio": func(w http.ResponseWriter, r *http.Request) {
			w.Write(listJSON(t, []map[string]interface{}{
				{"asset_id": "a1", "balance": map[string]string{"value": "2.0"}, "available_balance": map[string]string{"value": "1.5"}, "currency_balance": map[string]string{"value": "200000"}},
				{"asset_id": "a2", "balance": map[string]string{"value": "10.0"}, "available_balance": map[string]string{"value": "10.0"}, "currency_balance": map[string]string{"value": "30000"}},
			}))
		},
		"/v1/assets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "a1", "name": "Bitcoin", "symbol": "BTC"},
				{"id": "a2", "name": "Ethereum", "symbol": "ETH"},
			}))
		},
		"/v1/currencies": func(w http.ResponseWriter, r *http.Request) {
			w.Write(listJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "value", "", "", "")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	// 2 assets, sorted by value descending => BTC first. The TOTAL summary row
	// must NOT appear in JSON output (it is a table-only footer).
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d: %v", len(rows), rows)
	}
	if rows[0]["Asset"] != "BTC" {
		t.Errorf("expected BTC first (highest value), got %s", rows[0]["Asset"])
	}
	for _, r := range rows {
		if r["Asset"] == "TOTAL" {
			t.Errorf("TOTAL row leaked into JSON output: %v", r)
		}
	}
	// Available balance should be surfaced as its own column.
	if _, ok := rows[0]["Available"]; !ok {
		t.Errorf("expected Available column, got: %v", rows[0])
	}
}

func TestRunPortfolio_TotalRowOnlyInTable(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/portfolio": func(w http.ResponseWriter, r *http.Request) {
			w.Write(listJSON(t, []map[string]interface{}{
				{"asset_id": "a1", "balance": map[string]string{"value": "2.0"}, "currency_balance": map[string]string{"value": "200000"}},
			}))
		},
		"/v1/assets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{{"id": "a1", "name": "Bitcoin", "symbol": "BTC"}}))
		},
		"/v1/currencies": func(w http.ResponseWriter, r *http.Request) {
			w.Write(listJSON(t, []map[string]string{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	app.outFormat = output.FormatTable
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runPortfolio(cmd, "value", "", "", "")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if !strings.Contains(raw, "TOTAL") {
		t.Errorf("expected TOTAL footer in table output, got:\n%s", raw)
	}
}

func TestRunPrice_ResolvesSymbol(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/assets": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("symbol"); got != "BTC" {
				t.Errorf("symbol = %q, want BTC", got)
			}
			w.Write(paginatedJSON(t, []map[string]string{{"id": "a1", "name": "Bitcoin", "symbol": "BTC"}}))
		},
		"/v1/tickers/": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/tickers/a1" {
				t.Errorf("path = %q, want /v1/tickers/a1", r.URL.Path)
			}
			w.Write([]byte(`{"data":{"asset_id":"a1","price":"50000","currency_id":"eur"}}`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runPrice(cmd, []string{"btc"})
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 || rows[0]["Price"] != "50000" || rows[0]["Symbol"] != "BTC" {
		t.Fatalf("unexpected rows: %v", rows)
	}
}

func TestRunOperations_FlattensTransactions(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/operations": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{
					"operation_id":   "op1",
					"operation_type": "buy",
					"transactions": []map[string]interface{}{
						{"transaction_id": "tx1", "asset_id": "a1", "flow": "INCOMING", "asset_amount": map[string]string{"value": "1.0"}, "trade": map[string]string{"trade_id": "t1"}},
					},
				},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runOperations(cmd, nil, nil, "", "", 0, 25, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %v", len(rows), rows)
	}
	if rows[0]["Flow"] != "incoming" || rows[0]["Trade ID"] != "t1" || rows[0]["Amount"] != "1.0" {
		t.Errorf("unexpected row: %v", rows[0])
	}
}

func TestRunTrades_OnlyTradeTransactions(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/operations": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{
					"operation_id":   "op1",
					"operation_type": "buy",
					"transactions": []map[string]interface{}{
						{"transaction_id": "tx1", "asset_id": "a1", "asset_amount": map[string]string{"value": "1.0"}, "trade": map[string]string{"trade_id": "t1", "rate": "50000"}},
						{"transaction_id": "tx2", "currency_id": "eur", "asset_amount": map[string]string{"value": "-50000"}},
					},
				},
			}))
		},
		"/v1/assets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin"}}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runTrades(cmd, "", "", "", 0, false)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 trade row (only tx with trade), got %d: %v", len(rows), rows)
	}
	if rows[0]["Symbol"] != "BTC" || rows[0]["Rate"] != "50000" {
		t.Errorf("unexpected row: %v", rows[0])
	}
}

func TestRunCurrencies(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/currencies": func(w http.ResponseWriter, r *http.Request) {
			w.Write(listJSON(t, []map[string]string{{"id": "c1", "symbol": "EUR", "name": "Euro"}}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runCurrencies(cmd, "")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 || rows[0]["Symbol"] != "EUR" {
		t.Fatalf("unexpected rows: %v", rows)
	}
}

func TestRunQuoteCreate_RejectsNonNumericAmountBeforeAPICall(t *testing.T) {
	called := false
	server := newMockServer(t, mockEndpoints{
		"/v1/quotes": func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.Write([]byte(`{"data":{"quote_id":"q1"}}`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	// ".5" is not a valid JSON number; it must be rejected up front.
	err := app.runQuoteCreate(cmd, "a1", "c1", "buy", ".5", "")
	if err == nil {
		t.Fatal("expected validation error for --quantity \".5\", got nil")
	}
	if !strings.Contains(err.Error(), "quantity") {
		t.Errorf("error should name the quantity flag, got: %v", err)
	}
	if called {
		t.Error("API was called despite invalid --quantity")
	}
}

func TestRunQuoteAccept_AbortsWithoutConfirmation(t *testing.T) {
	called := false
	server := newMockServer(t, mockEndpoints{
		"/v1/quotes/": func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.Write([]byte(`{"data":{"quote_id":"q1","trade_id":"t1"}}`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	app.assumeYes = false
	cmd := newTestCmd()
	cmd.SetIn(strings.NewReader("n\n"))
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)

	if err := app.runQuoteAccept(cmd, "q1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected accept to be aborted, but the API was called")
	}
	if !strings.Contains(errBuf.String(), "Aborted") {
		t.Errorf("expected Aborted message, got %q", errBuf.String())
	}
}

func TestValidateAmount(t *testing.T) {
	valid := []string{"1.5", "0.5", "1", "1e3", "0.00000001", "100000"}
	for _, amt := range valid {
		if err := validateAmount("amount", amt); err != nil {
			t.Errorf("validateAmount(%q) = %v, want nil", amt, err)
		}
	}

	invalid := []string{"", ".5", "1,5", "NaN", "Inf", "abc", "0", "-1", "00.5", "0x1"}
	for _, amt := range invalid {
		if err := validateAmount("amount", amt); err == nil {
			t.Errorf("validateAmount(%q) = nil, want error", amt)
		}
	}
}

func TestRunEarnAction_RejectsInvalidAmountBeforeAPICall(t *testing.T) {
	called := false
	server := newMockServer(t, mockEndpoints{
		"/v1/earn/actions/": func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.Write([]byte(`{"data":{"status":"INITIATED"}}`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	app.assumeYes = true
	cmd := newTestCmd()

	// ".5" passes a naive ParseFloat but is not a valid JSON number; it must be
	// rejected up front rather than failing later during request encoding.
	err := app.runEarnAction(cmd, "STAKE", "cfg1", "", ".5")
	if err == nil {
		t.Fatal("expected validation error for amount \".5\", got nil")
	}
	if called {
		t.Error("API was called despite invalid amount")
	}
}

func TestRunEarnAction_ExecutesWithYes(t *testing.T) {
	var gotPath string
	server := newMockServer(t, mockEndpoints{
		"/v1/earn/actions/": func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Write([]byte(`{"data":{"status":"INITIATED"}}`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	app.assumeYes = true
	cmd := newTestCmd()

	var runErr error
	raw := captureStdout(t, func() {
		runErr = app.runEarnAction(cmd, "STAKE", "cfg1", "", "1.5")
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if gotPath != "/v1/earn/actions/STAKE" {
		t.Errorf("path = %q, want /v1/earn/actions/STAKE", gotPath)
	}
	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 || rows[0]["Status"] != "INITIATED" {
		t.Fatalf("unexpected rows: %v", rows)
	}
}
