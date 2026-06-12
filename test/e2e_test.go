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
	for _, want := range []string{"portfolio", "operations", "trades", "currencies", "earn", "quote", "--output"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("help output missing %q", want)
		}
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
	_, stderr, code := runBP(t, []string{"BITPANDA_API_KEY=", "HOME=/nonexistent"}, "portfolio")
	if code == 0 {
		t.Error("expected non-zero exit code for missing API key")
	}
	if !strings.Contains(stderr, "no API key found") && !strings.Contains(stderr, "API key") {
		t.Errorf("expected API key error message, got: %s", stderr)
	}
}

// mockServer creates a test HTTP server that mimics the Bitpanda Public API.
func mockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
			w.Write([]byte(`Unauthorized`))
			return
		}

		switch {
		case r.URL.Path == "/v1/portfolio":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"asset_id": "a1", "balance": map[string]string{"value": "1.5"}, "currency_balance": map[string]string{"value": "142500.00"}, "invested_amount": map[string]string{"value": "100000.00"}, "total_return": map[string]string{"value": "42500.00"}, "total_return_percent": "42.5"},
					{"asset_id": "a2", "balance": map[string]string{"value": "100.0"}, "currency_balance": map[string]string{"value": "350000.00"}},
				},
			})

		case r.URL.Path == "/v1/portfolio-history":
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"datapoints": []map[string]any{
						{"time": "2024-06-01T00:00:00Z", "value": map[string]string{"value": "490000.00"}},
					},
					"return_percentage": "12.3",
				},
			})

		case r.URL.Path == "/v1/assets":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "a1", "name": "Bitcoin", "symbol": "BTC", "type": "cryptocoin"},
					{"id": "a2", "name": "Ethereum", "symbol": "ETH", "type": "cryptocoin"},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		case r.URL.Path == "/v1/currencies":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "c1", "symbol": "EUR", "name": "Euro"},
				},
			})

		case strings.HasPrefix(r.URL.Path, "/v1/tickers/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/tickers/")
			prices := map[string]string{"a1": "95000.00", "a2": "3500.00"}
			price, ok := prices[id]
			if !ok {
				w.WriteHeader(404)
				w.Write([]byte(`{"error":{"code":"not_found"}}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"asset_id": id, "price": price, "currency_id": "c1"},
			})

		case r.URL.Path == "/v1/operations":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"operation_id":   "op1",
						"operation_type": "buy",
						"transactions": []map[string]any{
							{"transaction_id": "tx1", "asset_id": "a1", "flow": "INCOMING", "asset_amount": map[string]string{"value": "0.5"}, "fee_amount": map[string]string{"value": "0.001"}, "credited_at": "2024-06-01T12:00:00Z", "trade": map[string]any{"trade_id": "trade1", "rate": "95000.00", "to_eur_rate": "1"}},
						},
					},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		case r.URL.Path == "/v1/earn/configs":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "cfg1", "asset_id": "a1", "annual_percentage_rate": 5.5, "type": "STAKING", "mode": "FLEXIBLE", "enabled": true, "soldout": false},
				},
				"has_next_page": false,
				"next_cursor":   "",
			})

		case r.URL.Path == "/v1/quotes":
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"quote_id": "q1", "status": "OPEN", "side": "BUY", "quote": map[string]string{"price": "95000.00", "notional": "10", "expires_at": "2024-06-01T12:01:00Z"}},
			})

		case r.URL.Path == "/v1/quotes/q1/accept":
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"quote_id": "q1", "trade_id": "t1", "status": "FINISHED", "side": "BUY", "execution": map[string]string{"price": "95000.00", "executed_at": "2024-06-01T12:00:30Z"}},
			})

		case strings.HasPrefix(r.URL.Path, "/v1/earn/actions/"):
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"status": "INITIATED"},
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

func TestPortfolioHistoryE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "portfolio-history", "--timeframe", "month")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "490000.00") {
		t.Errorf("expected datapoint value in output, got: %s", stdout)
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
	if !strings.Contains(stdout, "95000.00") {
		t.Errorf("expected BTC price in prices output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "3500.00") {
		t.Errorf("expected ETH price in prices output, got: %s", stdout)
	}
}

func TestOperationsE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "operations")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "tx1") {
		t.Errorf("expected transaction ID in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "trade1") {
		t.Errorf("expected trade ID in output, got: %s", stdout)
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

func TestCurrenciesE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "currencies")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "EUR") {
		t.Errorf("expected EUR in currencies output, got: %s", stdout)
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

func TestAssetsListE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "assets")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Bitcoin") || !strings.Contains(stdout, "Ethereum") {
		t.Errorf("expected assets list, got: %s", stdout)
	}
}

func TestEarnConfigsE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "earn", "configs")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "cfg1") || !strings.Contains(stdout, "STAKING") {
		t.Errorf("expected earn config in output, got: %s", stdout)
	}
}

func TestEarnStakeWithYesE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "--yes", "earn", "stake", "--config-id", "cfg1", "--amount", "1.5")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "INITIATED") {
		t.Errorf("expected INITIATED status, got: %s", stdout)
	}
}

func TestQuoteCreateAndAcceptE2E(t *testing.T) {
	server := mockServer(t)
	defer server.Close()

	stdout, _, code := runBP(t, mockEnv(server.URL), "quote", "create", "--asset-id", "a1", "--currency-id", "c1", "--side", "buy", "--notional", "10")
	if code != 0 {
		t.Errorf("create exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "q1") {
		t.Errorf("expected quote id, got: %s", stdout)
	}

	stdout, _, code = runBP(t, mockEnv(server.URL), "--yes", "quote", "accept", "q1")
	if code != 0 {
		t.Errorf("accept exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "t1") {
		t.Errorf("expected trade id, got: %s", stdout)
	}
}

func TestAllCommandsHaveHelp(t *testing.T) {
	commands := [][]string{
		{"portfolio"}, {"portfolio-history"}, {"operations"}, {"trades"},
		{"price"}, {"prices"}, {"asset"}, {"assets"}, {"currencies"},
		{"earn"}, {"earn", "configs"}, {"earn", "stake"}, {"earn", "unstake"},
		{"quote"}, {"quote", "create"}, {"quote", "accept"},
	}
	for _, cmd := range commands {
		name := strings.Join(cmd, " ")
		t.Run(name, func(t *testing.T) {
			args := append(cmd, "--help")
			stdout, _, code := runBP(t, nil, args...)
			if code != 0 {
				t.Errorf("%s --help: exit code = %d, want 0", name, code)
			}
			if stdout == "" {
				t.Errorf("%s --help: no output", name)
			}
		})
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	_, stderr, code := runBP(t, []string{"BITPANDA_API_KEY=fake"}, "portfolio", "--output", "xml")
	if code == 0 {
		t.Error("expected non-zero exit code for invalid output format")
	}
	if !strings.Contains(stderr, "invalid output format") {
		t.Errorf("expected format error, got: %s", stderr)
	}
}
