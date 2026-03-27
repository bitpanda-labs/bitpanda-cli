package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_FlagTakesPriority(t *testing.T) {
	t.Setenv("BITPANDA_API_KEY", "env-key")

	cfg, err := Load("flag-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "flag-key" {
		t.Errorf("expected flag-key, got %s", cfg.APIKey)
	}
}

func TestLoad_EnvFallback(t *testing.T) {
	t.Setenv("BITPANDA_API_KEY", "env-key")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("expected env-key, got %s", cfg.APIKey)
	}
}

func TestLoad_ConfigFileFallback(t *testing.T) {
	// Create a temp home dir with config file
	tmpHome := t.TempDir()
	cfgDir := filepath.Join(tmpHome, ".config", "bitpanda")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("api_key: file-key\n"), 0o644)

	t.Setenv("HOME", tmpHome)
	t.Setenv("BITPANDA_API_KEY", "")
	os.Unsetenv("BITPANDA_API_KEY")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("expected file-key, got %s", cfg.APIKey)
	}
}

func TestLoad_NoKeyReturnsError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("BITPANDA_API_KEY", "")
	os.Unsetenv("BITPANDA_API_KEY")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error when no API key is available")
	}
}

func TestLoad_BaseURLFromEnv(t *testing.T) {
	t.Setenv("BITPANDA_API_KEY", "some-key")
	t.Setenv("BITPANDA_BASE_URL", "http://localhost:8080")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("expected base URL http://localhost:8080, got %s", cfg.BaseURL)
	}
}

func TestCheckConfigFilePermissions_SecureFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(tmpFile, []byte("api_key: test\n"), 0o600)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	checkConfigFilePermissions(tmpFile)

	w.Close()
	os.Stderr = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	r.Close()

	if n > 0 {
		t.Errorf("expected no warning for 0600 permissions, got: %s", string(buf[:n]))
	}
}

func TestCheckConfigFilePermissions_InsecureFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(tmpFile, []byte("api_key: test\n"), 0o644)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	checkConfigFilePermissions(tmpFile)

	w.Close()
	os.Stderr = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	r.Close()

	output := string(buf[:n])
	if !strings.Contains(output, "Warning:") || !strings.Contains(output, "consider restricting to 0600") {
		t.Errorf("expected permission warning, got: %s", output)
	}
}
