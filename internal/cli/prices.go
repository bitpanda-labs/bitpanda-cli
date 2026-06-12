package cli

import (
	"fmt"
	"sort"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerPrices(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "prices",
		Short: "List prices for held assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPrices(cmd)
		},
	}

	parent.AddCommand(cmd)
}

func (app *App) runPrices(cmd *cobra.Command) error {
	ctx := cmd.Context()

	positions, err := app.apiClient.GetPortfolio(ctx, api.PortfolioParams{})
	if err != nil {
		return err
	}

	assets, err := app.apiClient.ListAllAssets(ctx)
	if err != nil {
		return fmt.Errorf("fetching assets: %w", err)
	}

	// Collect unique non-fiat asset IDs held in the portfolio.
	seen := make(map[string]bool)
	var assetIDs []string
	for _, p := range positions {
		if p.AssetID == "" || seen[p.AssetID] {
			continue
		}
		seen[p.AssetID] = true
		assetIDs = append(assetIDs, p.AssetID)
	}
	sort.Slice(assetIDs, func(i, j int) bool {
		return assets[assetIDs[i]].Symbol < assets[assetIDs[j]].Symbol
	})

	columns := []string{"Symbol", "Asset ID", "Price", "Currency ID"}
	rows := make([][]string, 0, len(assetIDs))
	for _, id := range assetIDs {
		ticker, err := app.apiClient.GetTicker(ctx, id)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: no ticker for asset %s: %v\n", id, err)
			continue
		}
		symbol := id
		if a, ok := assets[id]; ok {
			symbol = a.Symbol
		}
		rows = append(rows, []string{symbol, ticker.AssetID, ticker.Price, ticker.CurrencyID})
	}

	return output.Render(app.outFormat, columns, rows)
}
