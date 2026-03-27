package cli

import (
	"net/http"
	"testing"
)

func TestRunAsset_Found(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/assets/a1": func(w http.ResponseWriter, r *http.Request) {
			w.Write(assetJSON("a1", "Bitcoin", "BTC"))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	var raw string
	var runErr error
	raw = captureStdout(t, func() {
		runErr = app.runAsset(cmd, []string{"a1"})
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	rows := parseJSONOutput(t, raw)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["ID"] != "a1" {
		t.Errorf("ID = %q, want %q", rows[0]["ID"], "a1")
	}
	if rows[0]["Name"] != "Bitcoin" {
		t.Errorf("Name = %q, want %q", rows[0]["Name"], "Bitcoin")
	}
	if rows[0]["Symbol"] != "BTC" {
		t.Errorf("Symbol = %q, want %q", rows[0]["Symbol"], "BTC")
	}
}

func TestRunAsset_NotFound(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/assets/nonexistent": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte(`Not Found`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runAsset(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestRunAsset_APIError(t *testing.T) {
	server := newMockServer(t, mockEndpoints{
		"/v1/assets/a1": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte(`Internal Server Error`))
		},
	})
	defer server.Close()

	app := newTestApp(server.URL)
	cmd := newTestCmd()

	err := app.runAsset(cmd, []string{"a1"})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
