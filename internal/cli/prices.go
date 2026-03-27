package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func (app *App) registerPrices(parent *cobra.Command) {
	var all bool

	cmd := &cobra.Command{
		Use:   "prices",
		Short: "List prices for held assets (or --all for all available)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPrices(cmd, all)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show all available prices")
	parent.AddCommand(cmd)
}

func (app *App) runPrices(cmd *cobra.Command, all bool) error {
	ctx := cmd.Context()

	ticker, err := app.apiClient.FetchAllTicker(ctx)
	if err != nil {
		return err
	}

	var symbols []string

	if all {
		for s := range ticker {
			symbols = append(symbols, s)
		}
	} else {
		// Get held assets
		wallets, err := app.apiClient.ListWallets(ctx, api.WalletParams{PageSize: 100})
		if err != nil {
			return err
		}

		// Collect unique asset IDs with non-zero balance
		assetIDs := make(map[string]bool)
		for _, w := range wallets {
			bal, err := strconv.ParseFloat(w.Balance, 64)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: skipping wallet %s: invalid balance %q: %v\n", w.WalletID, w.Balance, err)
				continue
			}
			if bal > 0 {
				assetIDs[w.AssetID] = true
			}
		}

		// Resolve asset IDs to symbols
		seen := make(map[string]bool)
		for id := range assetIDs {
			asset, err := app.apiClient.GetAsset(ctx, id)
			if err != nil {
				continue
			}
			sym := asset.Data.Symbol
			if !seen[sym] {
				seen[sym] = true
				symbols = append(symbols, sym)
			}
		}
	}

	sort.Strings(symbols)

	columns := []string{"Symbol", "Price", "Currency", "24h Change"}
	rows := make([][]string, 0, len(symbols))
	for _, s := range symbols {
		entry, found := ticker[s]
		if !found {
			continue
		}
		rows = append(rows, []string{
			entry.Symbol,
			entry.Price,
			entry.Currency,
			entry.PriceChangeDay + "%",
		})
	}

	return output.Render(app.outFormat, columns, rows)
}
