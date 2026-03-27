package cli

import (
	"net/http"
	"testing"
)

func TestRunPrice_FoundSymbol(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "-1.5"},
				{"id": "2", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3000.00", "price_change_day": "2.3"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrice(cmd, []string{"BTC"})
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["Symbol"] != "BTC" {
		t.Errorf("Symbol = %q, want %q", rows[0]["Symbol"], "BTC")
	}
	if rows[0]["Price"] != "50000.00" {
		t.Errorf("Price = %q, want %q", rows[0]["Price"], "50000.00")
	}
	if rows[0]["Currency"] != "EUR" {
		t.Errorf("Currency = %q, want %q", rows[0]["Currency"], "EUR")
	}
	if rows[0]["24h Change"] != "-1.5%" {
		t.Errorf("24h Change = %q, want %q", rows[0]["24h Change"], "-1.5%")
	}
}

func TestRunPrice_SymbolNotFound(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runPrice(cmd, []string{"NONEXISTENT"})
	if err == nil {
		t.Fatal("expected error for unknown symbol")
	}
	if err.Error() != `symbol "NONEXISTENT" not found` {
		t.Errorf("error = %q, want %q", err.Error(), `symbol "NONEXISTENT" not found`)
	}
}

func TestRunPrice_CaseInsensitive(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"id": "1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "50000.00", "price_change_day": "0"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runPrice(cmd, []string{"btc"})
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v (lowercase input should match uppercase symbol)", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["Symbol"] != "BTC" {
		t.Errorf("Symbol = %q, want %q", rows[0]["Symbol"], "BTC")
	}
}

func TestRunPrice_APIError(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/ticker": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte(`Internal Server Error`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runPrice(cmd, []string{"BTC"})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
