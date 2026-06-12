// Package cli defines the Cobra command tree for the bp CLI, wiring together
// configuration, API calls, and output formatting for each subcommand.
package cli

import (
	"fmt"
	"os"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/config"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

// Exit code constants for structured error reporting.
const (
	ExitGeneral = 1 // general error
	ExitAuth    = 2 // authentication error (401)
	ExitAPI     = 3 // other API error
)

var Version = "0.1.0"

// App holds the shared state for the CLI, replacing package-level globals.
type App struct {
	cfg       *config.Config
	apiClient *api.Client
	outFormat output.Format
	assumeYes bool // skip confirmation prompts for destructive actions
}

func newApp() *cobra.Command {
	app := &App{}

	var flagAPIKey string
	var flagOutput string
	var flagInsecure bool
	var flagYes bool

	rootCmd := &cobra.Command{
		Use:   "bp",
		Short: "Bitpanda CLI — interact with your Bitpanda account from the terminal",
		Long: `bp is a command-line tool for the Bitpanda Public API.

View your portfolio, check prices, browse trades and operations, and
place trades — all from your terminal. Supports table, JSON, and CSV output.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse output format
			f, err := output.ParseFormat(flagOutput)
			if err != nil {
				return err
			}
			app.outFormat = f
			app.assumeYes = flagYes

			// Skip config loading for help/version/completion
			if cmd.Name() == "help" || cmd.Name() == "bp" || cmd.Name() == "completion" ||
				cmd.Name() == cobra.ShellCompRequestCmd || cmd.Name() == cobra.ShellCompNoDescRequestCmd {
				return nil
			}

			// Load config and create API client
			c, err := config.Load(flagAPIKey)
			if err != nil {
				return err
			}
			app.cfg = c
			app.apiClient = api.NewClient(app.cfg.APIKey, flagInsecure)
			if app.cfg.BaseURL != "" {
				app.apiClient.BaseURL = app.cfg.BaseURL
			}
			app.apiClient.SetUserAgent(Version)
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, json, csv")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "Bitpanda API key (overrides env and config file)")
	rootCmd.PersistentFlags().BoolVar(&flagInsecure, "insecure", false, "Skip TLS certificate verification")
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation prompts for trading and earn actions")
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("bp version {{.Version}}\n")

	app.registerPortfolio(rootCmd)
	app.registerPortfolioHistory(rootCmd)
	app.registerOperations(rootCmd)
	app.registerTrades(rootCmd)
	app.registerPrice(rootCmd)
	app.registerPrices(rootCmd)
	app.registerAsset(rootCmd)
	app.registerAssets(rootCmd)
	app.registerCurrencies(rootCmd)
	app.registerEarn(rootCmd)
	app.registerQuote(rootCmd)
	app.registerCompletion(rootCmd)

	return rootCmd
}

// Execute runs the root command. It returns an error (possibly an *ExitError
// carrying a specific exit code) instead of calling os.Exit directly, so that
// the caller (main) controls process termination.
func Execute() error {
	rootCmd := newApp()
	err := rootCmd.Execute()
	if err == nil {
		return nil
	}

	fmt.Fprintln(os.Stderr, "Error:", err)

	// Map API errors to specific exit codes
	if apiErr, ok := err.(*api.APIError); ok {
		if apiErr.IsAuthError() {
			return newExitError(ExitAuth, err)
		}
		return newExitError(ExitAPI, err)
	}

	return err
}
