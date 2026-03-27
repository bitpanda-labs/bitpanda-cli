package cli

import (
	"net/http"
	"strings"
	"testing"

)

func TestRunBalances_NonZeroFilter(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.5", "last_credited_at": "2024-06-15"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "0.0", "last_credited_at": "2024-06-10"},
				{"wallet_id": "w3", "asset_id": "a3", "wallet_type": "", "balance": "10.0", "last_credited_at": "2024-06-12"},
				{"wallet_id": "w4", "asset_id": "a4", "wallet_type": "", "balance": "0", "last_credited_at": "2024-06-08"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runBalances(cmd, "", true, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 non-zero wallets, got %d", len(rows))
	}

	walletIDs := make(map[string]bool)
	for _, row := range rows {
		walletIDs[row["Wallet ID"]] = true
	}
	if !walletIDs["w1"] || !walletIDs["w3"] {
		t.Errorf("expected wallets w1 and w3, got %v", walletIDs)
	}
}

func TestRunBalances_NonZeroFilterRemovesAll(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "0.0", "last_credited_at": "2024-06-15"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "0", "last_credited_at": "2024-06-10"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runBalances(cmd, "", true, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows when non-zero removes all, got %d", len(rows))
	}
}

func TestRunBalances_WithoutNonZeroFilter(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.5", "last_credited_at": "2024-06-15"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "0.0", "last_credited_at": "2024-06-10"},
				{"wallet_id": "w3", "asset_id": "a3", "wallet_type": "", "balance": "10.0", "last_credited_at": "2024-06-12"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		// nonZero=false, should include all wallets
		runErr = app.runBalances(cmd, "", false, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 3 {
		t.Fatalf("expected 3 wallets (no filter), got %d", len(rows))
	}
}

func TestRunBalances_EmptyWalletTypeDefaultsToRegular(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "1.0", "last_credited_at": "2024-06-15"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "STAKING", "balance": "2.0", "last_credited_at": "2024-06-10"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runBalances(cmd, "", false, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// w1 has empty wallet_type, should be displayed as "regular"
	if rows[0]["Wallet Type"] != "regular" {
		t.Errorf("expected wallet type 'regular' for empty type, got %s", rows[0]["Wallet Type"])
	}
	// w2 keeps its explicit type
	if rows[1]["Wallet Type"] != "STAKING" {
		t.Errorf("expected wallet type 'STAKING', got %s", rows[1]["Wallet Type"])
	}
}

func TestRunBalances_EmptyWallets(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
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
		runErr = app.runBalances(cmd, "", false, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for empty wallets, got %d", len(rows))
	}
}

func TestRunBalances_InvalidBalanceSkippedInNonZeroFilter(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "not-a-number", "last_credited_at": "2024-06-15"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "5.0", "last_credited_at": "2024-06-10"},
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
		runErr = app.runBalances(cmd, "", true, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (invalid balance skipped), got %d", len(rows))
	}
	if rows[0]["Wallet ID"] != "w2" {
		t.Errorf("expected wallet w2, got %s", rows[0]["Wallet ID"])
	}

	if !strings.Contains(stderrBuf.String(), "Warning: skipping wallet w1") {
		t.Errorf("expected warning about invalid balance, got: %s", stderrBuf.String())
	}
}

func TestRunBalances_SingleWallet(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "SAVINGS", "balance": "42.5", "last_credited_at": "2024-12-25"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runBalances(cmd, "", false, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["Wallet ID"] != "w1" {
		t.Errorf("wallet_id = %s, want w1", row["Wallet ID"])
	}
	if row["Asset ID"] != "a1" {
		t.Errorf("asset_id = %s, want a1", row["Asset ID"])
	}
	if row["Wallet Type"] != "SAVINGS" {
		t.Errorf("wallet_type = %s, want SAVINGS", row["Wallet Type"])
	}
	if row["Balance"] != "42.5" {
		t.Errorf("balance = %s, want 42.5", row["Balance"])
	}
	if row["Last Credited"] != "2024-12-25" {
		t.Errorf("last_credited = %s, want 2024-12-25", row["Last Credited"])
	}
}

func TestRunBalances_NonZeroWithNegativeBalance(t *testing.T) {
	// Negative balances are not > 0, so they should be filtered by --non-zero.
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "-1.0", "last_credited_at": "2024-06-15"},
				{"wallet_id": "w2", "asset_id": "a2", "wallet_type": "", "balance": "5.0", "last_credited_at": "2024-06-10"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runBalances(cmd, "", true, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row (negative balance filtered), got %d", len(rows))
	}
	if rows[0]["Wallet ID"] != "w2" {
		t.Errorf("expected wallet w2, got %s", rows[0]["Wallet ID"])
	}
}

func TestRunBalances_AssetIDFilter(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("asset_id"); got != "a1" {
				t.Errorf("asset_id = %q, want %q", got, "a1")
			}
			w.Write(paginatedJSON(t, []map[string]string{
				{"wallet_id": "w1", "asset_id": "a1", "wallet_type": "", "balance": "5.0", "last_credited_at": "2024-06-15"},
			}))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runBalances(cmd, "a1", false, 0, 100)
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["Asset ID"] != "a1" {
		t.Errorf("expected asset a1, got %s", rows[0]["Asset ID"])
	}
}

func TestRunBalances_APIError(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			w.Write([]byte(`Unauthorized`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runBalances(cmd, "", false, 0, 25)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
