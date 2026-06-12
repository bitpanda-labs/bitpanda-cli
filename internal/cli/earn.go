package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerEarn(parent *cobra.Command) {
	earnCmd := &cobra.Command{
		Use:   "earn",
		Short: "View earn configurations and stake or unstake assets",
	}

	app.registerEarnConfigs(earnCmd)
	app.registerEarnAction(earnCmd, "stake", "STAKE")
	app.registerEarnAction(earnCmd, "unstake", "UNSTAKE")

	parent.AddCommand(earnCmd)
}

func (app *App) registerEarnConfigs(parent *cobra.Command) {
	var (
		assetID  string
		typ      string
		mode     string
		limit    int
		pageSize int
		all      bool
	)

	cmd := &cobra.Command{
		Use:   "configs",
		Short: "List earn asset configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runEarnConfigs(cmd, assetID, typ, mode, limit, pageSize, all)
		},
	}

	cmd.Flags().StringVar(&assetID, "asset-id", "", "Filter by asset UUID")
	cmd.Flags().StringVar(&typ, "type", "", "Filter by earn type")
	cmd.Flags().StringVar(&mode, "mode", "", "Filter by earn mode")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Items per API page")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages")
	parent.AddCommand(cmd)
}

func (app *App) runEarnConfigs(cmd *cobra.Command, assetID, typ, mode string, limit, pageSize int, all bool) error {
	fetchLimit := limit
	if !all && fetchLimit == 0 {
		fetchLimit = pageSize
	}

	configs, err := app.apiClient.ListEarnConfigs(cmd.Context(), api.EarnConfigParams{
		AssetID:  assetID,
		Type:     typ,
		Mode:     mode,
		PageSize: pageSize,
		Limit:    fetchLimit,
	})
	if err != nil {
		return err
	}

	columns := []string{"Config ID", "Asset ID", "APR", "Type", "Mode", "Enabled", "Soldout"}
	rows := make([][]string, 0, len(configs))
	for _, c := range configs {
		rows = append(rows, []string{
			c.ID,
			c.AssetID,
			strconv.FormatFloat(c.AnnualPercentageRate, 'f', -1, 64),
			c.Type,
			c.Mode,
			strconv.FormatBool(c.Enabled),
			strconv.FormatBool(c.Soldout),
		})
	}

	return output.Render(app.outFormat, columns, rows)
}

// registerEarnAction registers a stake or unstake command. apiAction is the
// uppercase action sent to the API (STAKE/UNSTAKE).
func (app *App) registerEarnAction(parent *cobra.Command, name, apiAction string) {
	var (
		configID string
		walletID string
		amount   string
	)

	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s an asset", cases(name)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runEarnAction(cmd, apiAction, configID, walletID, amount)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "Earn config UUID (required)")
	cmd.Flags().StringVar(&walletID, "wallet-id", "", "Wallet UUID")
	cmd.Flags().StringVar(&amount, "amount", "", "Asset amount (required)")
	_ = cmd.MarkFlagRequired("config-id")
	_ = cmd.MarkFlagRequired("amount")
	parent.AddCommand(cmd)
}

func (app *App) runEarnAction(cmd *cobra.Command, apiAction, configID, walletID, amount string) error {
	if err := validateAmount("amount", amount); err != nil {
		return err
	}

	prompt := fmt.Sprintf("%s %s of config %s?", cases(apiAction), amount, configID)
	if !app.assumeYes && !confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), prompt) {
		fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
		return nil
	}

	result, err := app.apiClient.ExecuteEarnAction(cmd.Context(), apiAction, api.EarnActionRequest{
		ConfigID:    configID,
		WalletID:    walletID,
		AssetAmount: amount,
	})
	if err != nil {
		return err
	}

	columns := []string{"Action", "Config ID", "Amount", "Status"}
	rows := [][]string{{apiAction, configID, amount, result.Status}}
	return output.Render(app.outFormat, columns, rows)
}

// validateAmount checks that value is a positive, non-zero decimal that is also
// a valid JSON number literal. The literal check matches what the API layer sends
// (json.Number), so inputs like ".5", "1,5", or "NaN" are rejected here with a
// clear message instead of failing later during request encoding. The server
// additionally requires a positive value, enforced here too. flag is the name of
// the originating CLI flag (e.g. "amount", "quantity") used in error messages.
func validateAmount(flag, value string) error {
	if !json.Valid([]byte(value)) {
		return fmt.Errorf("invalid --%s %q: must be a number (e.g. 1.5, not .5)", flag, value)
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid --%s %q: must be a number", flag, value)
	}
	if v <= 0 {
		return fmt.Errorf("invalid --%s %q: must be greater than zero", flag, value)
	}
	return nil
}

// cases title-cases a lowercase/uppercase word for display (e.g. "stake" -> "Stake").
func cases(s string) string {
	if s == "" {
		return s
	}
	lower := []rune(s)
	for i, r := range lower {
		if i == 0 {
			if r >= 'a' && r <= 'z' {
				lower[i] = r - 32
			}
		} else if r >= 'A' && r <= 'Z' {
			lower[i] = r + 32
		}
	}
	return string(lower)
}
