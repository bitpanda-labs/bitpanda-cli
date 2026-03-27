package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary
	dir, _ := os.Getwd()
	binaryPath = filepath.Join(dir, "bp-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/bp")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %s\n%s\n", err, out)
		os.Exit(1)
	}

	code := m.Run()
	os.Remove(binaryPath)
	os.Exit(code)
}

func runBP(t *testing.T, env []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run binary: %v", err)
	}

	return stdout.String(), stderr.String(), exitCode
}

func TestHelp(t *testing.T) {
	stdout, _, code := runBP(t, nil, "--help")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "portfolio") {
		t.Error("help output missing 'portfolio' command")
	}
	if !strings.Contains(stdout, "trades") {
		t.Error("help output missing 'trades' command")
	}
	if !strings.Contains(stdout, "--output") {
		t.Error("help output missing '--output' flag")
	}
}

func TestVersion(t *testing.T) {
	stdout, _, code := runBP(t, nil, "--version")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.HasPrefix(stdout, "bp version ") {
		t.Errorf("version output = %q, want prefix 'bp version '", stdout)
	}
}

func TestMissingAPIKey(t *testing.T) {
	_, stderr, code := runBP(t, []string{"BITPANDA_API_KEY=", "HOME=/nonexistent"}, "balances")
	if code == 0 {
		t.Error("expected non-zero exit code for missing API key")
	}
	if !strings.Contains(stderr, "no API key found") && !strings.Contains(stderr, "API key") {
		t.Errorf("expected API key error message, got: %s", stderr)
	}
}

// mockServer creates a test HTTP server that mimics the Bitpanda API.
func mockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
			w.Write([]byte(`Unauthorized`))
			return
		}

		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/wallets"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"wallet_id":       "w1",
						"asset_id":        "a1",
						"wallet_type":     "",
						"balance":         "1.5",
						"last_credited_at": "2024-06-15T10:00:00Z",
					},
					{
						"wallet_id":       "w2",
						"asset_id":        "a2",
						"wallet_type":     "STAKING",
						"balance":         "100.0",
						"last_credited_at": "2024-06-10T08:00:00Z",
					},
				},
				"has_next_page": false,
				"end_cursor":    "",
			})

		case r.URL.Path == "/v1/assets":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "a1", "name": "Bitcoin", "symbol": "BTC"},
					{"id": "a2", "name": "Ethereum", "symbol": "ETH"},
				},
				"has_next_page": false,
				"end_cursor":    "",
			})

		case strings.HasPrefix(r.URL.Path, "/v1/assets/a1"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"id": "a1", "name": "Bitcoin", "symbol": "BTC"},
			})

		case strings.HasPrefix(r.URL.Path, "/v1/assets/a2"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"id": "a2", "name": "Ethereum", "symbol": "ETH"},
			})

		case strings.HasPrefix(r.URL.Path, "/v1/ticker"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "t1", "symbol": "BTC", "type": "cryptocoin", "currency": "EUR", "price": "95000.00", "price_change_day": "1.5"},
					{"id": "t2", "symbol": "ETH", "type": "cryptocoin", "currency": "EUR", "price": "3500.00", "price_change_day": "-0.8"},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		case strings.HasPrefix(r.URL.Path, "/v1/transactions"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"transaction_id": "tx1",
						"asset_id":       "a1",
						"wallet_id":      "w1",
						"asset_amount":   "0.5",
						"fee_amount":     "0.001",
						"operation_type": "buy",
						"flow":           "incoming",
						"credited_at":    "2024-06-01T12:00:00Z",
						"trade_id":       "trade1",
					},
				},
				"has_next_page": false,
				"end_cursor":    "",
			})

		default:
			w.WriteHeader(404)
			w.Write([]byte(`Not Found`))
		}
	}))
}

// mockEnv returns environment variables that point the CLI at the mock server.
func mockEnv(serverURL string) []string {
	return []string{
		"BITPANDA_API_KEY=test-key",
		"BITPANDA_BASE_URL=" + serverURL,
	}
}

func TestBalancesE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "1.5") {
		t.Errorf("expected balance 1.5 in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "100.0") {
		t.Errorf("expected balance 100.0 in output, got: %s", stdout)
	}
}

func TestBalancesJSONOutput(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "balances", "--output", "json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "1.5") {
		t.Errorf("expected balance in JSON output, got: %s", stdout)
	}
}

func TestPriceE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "price", "BTC")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "95000.00") {
		t.Errorf("expected BTC price in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "BTC") {
		t.Errorf("expected BTC symbol in output, got: %s", stdout)
	}
}

func TestPricesE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "prices")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "BTC") {
		t.Errorf("expected BTC in prices output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "ETH") {
		t.Errorf("expected ETH in prices output, got: %s", stdout)
	}
}

func TestPortfolioE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "portfolio")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Bitcoin") {
		t.Errorf("expected Bitcoin in portfolio output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "TOTAL") {
		t.Errorf("expected TOTAL row in portfolio output, got: %s", stdout)
	}
}

func TestTradesE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "trades")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "buy") {
		t.Errorf("expected buy operation in trades output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Bitcoin") {
		t.Errorf("expected Bitcoin in trades output, got: %s", stdout)
	}
}

func TestTransactionsE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "transactions")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "tx1") {
		t.Errorf("expected transaction ID in output, got: %s", stdout)
	}
}

func TestAssetE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "asset", "a1")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Bitcoin") {
		t.Errorf("expected Bitcoin in asset output, got: %s", stdout)
	}
}

func TestAllCommandsHaveHelp(t *testing.T) {
	commands := []string{"portfolio", "balances", "trades", "transactions", "price", "prices", "asset"}
	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			stdout, _, code := runBP(t, nil, cmd, "--help")
			if code != 0 {
				t.Errorf("%s --help: exit code = %d, want 0", cmd, code)
			}
			if stdout == "" {
				t.Errorf("%s --help: no output", cmd)
			}
		})
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	_, stderr, code := runBP(t, []string{"BITPANDA_API_KEY=fake"}, "balances", "--output", "xml")
	if code == 0 {
		t.Error("expected non-zero exit code for invalid output format")
	}
	if !strings.Contains(stderr, "invalid output format") {
		t.Errorf("expected format error, got: %s", stderr)
	}
}
