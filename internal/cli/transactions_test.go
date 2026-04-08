package cli

import (
	"net/http"
	"testing"
)

func TestRunTransactions_BasicList(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{
					"transaction_id": "tx1", "asset_id": "a1", "operation_type": "buy",
					"flow": "incoming", "asset_amount": "1.5", "fee_amount": "0.01",
					"credited_at": "2024-01-15", "trade_id": "t1",
				},
				{
					"transaction_id": "tx2", "asset_id": "a2", "operation_type": "deposit",
					"flow": "incoming", "asset_amount": "100.0", "fee_amount": "0",
					"credited_at": "2024-01-20", "trade_id": "",
				},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTransactions(cmd, "", "", "", "", "", 0, 25, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(rows))
	}
	if rows[0]["Transaction ID"] != "tx1" {
		t.Errorf("expected tx1, got %s", rows[0]["Transaction ID"])
	}
	if rows[0]["Amount"] != "1.5" {
		t.Errorf("expected amount 1.5, got %s", rows[0]["Amount"])
	}
	if rows[0]["Fee"] != "0.01" {
		t.Errorf("expected fee 0.01, got %s", rows[0]["Fee"])
	}
	if rows[1]["Operation"] != "deposit" {
		t.Errorf("expected deposit, got %s", rows[1]["Operation"])
	}
}

func TestRunTransactions_EmptyList(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTransactions(cmd, "", "", "", "", "", 0, 25, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestRunTransactions_FilterParamsForwarded(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if got := q.Get("wallet_id"); got != "w1" {
				t.Errorf("wallet_id = %q, want %q", got, "w1")
			}
			if got := q.Get("flow"); got != "incoming" {
				t.Errorf("flow = %q, want %q", got, "incoming")
			}
			if got := q.Get("asset_id"); got != "a1" {
				t.Errorf("asset_id = %q, want %q", got, "a1")
			}
			if got := q.Get("from_including"); got != "2024-01-01" {
				t.Errorf("from_including = %q, want %q", got, "2024-01-01")
			}
			if got := q.Get("to_excluding"); got != "2024-02-01" {
				t.Errorf("to_excluding = %q, want %q", got, "2024-02-01")
			}
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{
					"transaction_id": "tx1", "asset_id": "a1", "operation_type": "buy",
					"flow": "incoming", "asset_amount": "1.0", "fee_amount": "0",
					"credited_at": "2024-01-15", "trade_id": "t1",
				},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTransactions(cmd, "w1", "incoming", "a1", "2024-01-01", "2024-02-01", 0, 25, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestRunTransactions_OutputColumns(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]interface{}{
				{
					"transaction_id": "tx1", "asset_id": "a1", "operation_type": "buy",
					"flow": "incoming", "asset_amount": "2.5", "fee_amount": "0.05",
					"credited_at": "2024-03-15T10:30:00Z", "trade_id": "trade-1",
				},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runTransactions(cmd, "", "", "", "", "", 0, 25, true)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["Transaction ID"] != "tx1" {
		t.Errorf("Transaction ID = %q, want %q", row["Transaction ID"], "tx1")
	}
	if row["Asset ID"] != "a1" {
		t.Errorf("Asset ID = %q, want %q", row["Asset ID"], "a1")
	}
	if row["Operation"] != "buy" {
		t.Errorf("Operation = %q, want %q", row["Operation"], "buy")
	}
	if row["Flow"] != "incoming" {
		t.Errorf("Flow = %q, want %q", row["Flow"], "incoming")
	}
	if row["Amount"] != "2.5" {
		t.Errorf("Amount = %q, want %q", row["Amount"], "2.5")
	}
	if row["Fee"] != "0.05" {
		t.Errorf("Fee = %q, want %q", row["Fee"], "0.05")
	}
	if row["Date"] != "2024-03-15T10:30:00Z" {
		t.Errorf("Date = %q, want %q", row["Date"], "2024-03-15T10:30:00Z")
	}
	if row["Trade ID"] != "trade-1" {
		t.Errorf("Trade ID = %q, want %q", row["Trade ID"], "trade-1")
	}
}

func TestRunTransactions_APIError(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/transactions": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte(`Internal Server Error`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runTransactions(cmd, "", "", "", "", "", 0, 25, true)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
