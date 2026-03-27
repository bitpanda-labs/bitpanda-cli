// Package config handles API key resolution with a three-tier priority:
// CLI flag, BITPANDA_API_KEY environment variable, and ~/.config/bitpanda/config.yaml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	APIKey  string
	BaseURL string // override API base URL (from BITPANDA_BASE_URL env var)
}

// Load resolves the API key with priority: flag > env > config file.
func Load(flagAPIKey string) (*Config, error) {
	baseURL := os.Getenv("BITPANDA_BASE_URL")

	if flagAPIKey != "" {
		return &Config{APIKey: flagAPIKey, BaseURL: baseURL}, nil
	}

	if envKey := os.Getenv("BITPANDA_API_KEY"); envKey != "" {
		return &Config{APIKey: envKey, BaseURL: baseURL}, nil
	}

	home, err := os.UserHomeDir()
	if err == nil {
		cfgPath := filepath.Join(home, ".config", "bitpanda")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(cfgPath)

		if err := viper.ReadInConfig(); err == nil {
			checkConfigFilePermissions(viper.ConfigFileUsed())

			if key := viper.GetString("api_key"); key != "" {
				return &Config{APIKey: key, BaseURL: baseURL}, nil
			}
		}
	}

	return nil, fmt.Errorf("no API key found. Provide one via:\n  --api-key flag\n  BITPANDA_API_KEY environment variable\n  ~/.config/bitpanda/config.yaml (api_key field)")
}

// checkConfigFilePermissions warns on stderr if the config file has permissions
// more permissive than 0600 (owner read/write only). Since the config file
// contains the API key, it should not be readable by group or others.
func checkConfigFilePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: config file %s has permissions %04o; consider restricting to 0600 (chmod 600 %s)\n", path, mode, path)
	}
}
