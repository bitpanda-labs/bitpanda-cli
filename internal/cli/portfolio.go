package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

type portfolioRow struct {
	AssetName   string
	AssetSymbol string
	Balance     float64
	EURPrice    float64
	EURValue    float64
	Wallets     map[string]float64 // wallet_type -> balance
}

func (app *App) registerPortfolio(parent *cobra.Command) {
	var sortFlag string

	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Show aggregated portfolio with EUR valuations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPortfolio(cmd, sortFlag)
		},
	}

	cmd.Flags().StringVar(&sortFlag, "sort", "name", "Sort by: name, value")
	parent.AddCommand(cmd)
}

func (app *App) runPortfolio(cmd *cobra.Command, sortFlag string) error {
	ctx := cmd.Context()

	// Fetch all non-zero wallets
	wallets, err := app.apiClient.ListWallets(ctx, api.WalletParams{PageSize: 100})
	if err != nil {
		return err
	}

	// Filter zero balances
	var nonZero []api.Wallet
	for _, w := range wallets {
		bal, err := strconv.ParseFloat(w.Balance, 64)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: skipping wallet %s: invalid balance %q: %v\n", w.WalletID, w.Balance, err)
			continue
		}
		if bal > 0 {
			nonZero = append(nonZero, w)
		}
	}

	if len(nonZero) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No assets with balance found.")
		return nil
	}

	// Resolve all asset names in a single batch request
	assetMap, err := app.apiClient.ListAllAssets(ctx)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not fetch assets: %v\n", err)
		assetMap = make(map[string]api.AssetData)
	}

	// Fetch ticker prices
	ticker, err := app.apiClient.FetchAllTicker(ctx)
	if err != nil {
		return fmt.Errorf("fetching prices: %w", err)
	}

	// Aggregate by asset
	agg := make(map[string]*portfolioRow)
	for _, w := range nonZero {
		bal, err := strconv.ParseFloat(w.Balance, 64)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: skipping wallet %s in aggregation: invalid balance %q: %v\n", w.WalletID, w.Balance, err)
			continue
		}
		asset, found := assetMap[w.AssetID]
		symbol := "unknown"
		name := "unknown"
		if found {
			symbol = asset.Symbol
			name = asset.Name
		}

		row, ok := agg[symbol]
		if !ok {
			price := 0.0
			if t, found := ticker[symbol]; found {
				var parseErr error
				price, parseErr = strconv.ParseFloat(t.Price, 64)
				if parseErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: invalid price %q for %s, using 0.00: %v\n", t.Price, symbol, parseErr)
					price = 0.0
				}
			}
			row = &portfolioRow{
				AssetName:   name,
				AssetSymbol: symbol,
				EURPrice:    price,
				Wallets:     make(map[string]float64),
			}
			agg[symbol] = row
		}

		row.Balance += bal
		row.EURValue = row.Balance * row.EURPrice

		wType := w.WalletType
		if wType == "" {
			wType = "regular"
		}
		row.Wallets[wType] += bal
	}

	// Sort
	rows := make([]*portfolioRow, 0, len(agg))
	for _, r := range agg {
		rows = append(rows, r)
	}

	switch sortFlag {
	case "value":
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].EURValue > rows[j].EURValue
		})
	default:
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].AssetName < rows[j].AssetName
		})
	}

	// Build output
	columns := []string{"Asset", "Symbol", "Balance", "EUR Price", "EUR Value"}
	tableRows := make([][]string, 0, len(rows)+1)
	totalEUR := 0.0
	for _, r := range rows {
		tableRows = append(tableRows, []string{
			r.AssetName,
			r.AssetSymbol,
			formatFloat(r.Balance),
			formatFloat(r.EURPrice),
			formatFloat(r.EURValue),
		})
		totalEUR += r.EURValue
	}

	// Add total row
	tableRows = append(tableRows, []string{"TOTAL", "", "", "", formatFloat(totalEUR)})

	return output.Render(app.outFormat, columns, tableRows)
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
