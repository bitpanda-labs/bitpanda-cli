package cli

import (
	"fmt"
	"strings"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerPrice(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "price <SYMBOL|UUID>",
		Short: "Get the current price for a single asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPrice(cmd, args)
		},
	}

	parent.AddCommand(cmd)
}

func (app *App) runPrice(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	arg := args[0]

	assetID := arg
	symbol := arg
	// If the argument is not a UUID, treat it as a symbol and resolve it.
	if !isUUID(arg) {
		assets, err := app.apiClient.ListAssets(ctx, api.AssetParams{Symbol: strings.ToUpper(arg), PageSize: 1})
		if err != nil {
			return err
		}
		if len(assets) == 0 {
			return fmt.Errorf("symbol %q not found", arg)
		}
		assetID = assets[0].ID
		symbol = assets[0].Symbol
	}

	ticker, err := app.apiClient.GetTicker(ctx, assetID)
	if err != nil {
		return err
	}

	columns := []string{"Symbol", "Asset ID", "Price", "Currency ID"}
	rows := [][]string{
		{symbol, ticker.AssetID, ticker.Price, ticker.CurrencyID},
	}

	return output.Render(app.outFormat, columns, rows)
}

// isUUID reports whether s looks like a UUID (8-4-4-4-12 hex).
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}
