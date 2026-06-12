package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func (app *App) registerTrades(parent *cobra.Command) {
	var (
		assetType string
		from      string
		to        string
		limit     int
		all       bool
	)

	cmd := &cobra.Command{
		Use:   "trades",
		Short: "Show buy/sell trade history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runTrades(cmd, assetType, from, to, limit, all)
		},
	}

	cmd.Flags().StringVar(&assetType, "asset-type", "", "Filter: cryptocoin, metal, stock, etf, commodity")
	cmd.Flags().StringVar(&from, "from", "", "From date (ISO 8601 date-time)")
	cmd.Flags().StringVar(&to, "to", "", "To date (ISO 8601 date-time)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of trades (0 = all fetched)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (may be slow)")
	parent.AddCommand(cmd)
}

func (app *App) runTrades(cmd *cobra.Command, assetType, from, to string, limit int, all bool) error {
	ctx := cmd.Context()

	// Default to fetching one page of operations unless --all is set.
	pageSize := 100
	fetchLimit := 0
	if !all {
		fetchLimit = pageSize
	}

	var progress io.Writer
	if all {
		if f, ok := cmd.ErrOrStderr().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			progress = f
		}
	}

	ops, err := app.apiClient.ListOperations(ctx, api.OperationParams{
		From:     from,
		To:       to,
		PageSize: pageSize,
		Limit:    fetchLimit,
		Progress: progress,
	})
	if err != nil {
		return err
	}

	// Enrich with asset metadata (name, symbol, type).
	assets, err := app.apiClient.ListAllAssets(ctx)
	if err != nil {
		return fmt.Errorf("fetching assets: %w", err)
	}

	columns := []string{"Date", "Operation", "Asset", "Symbol", "Type", "Amount", "Rate", "To EUR Rate", "Trade ID"}
	rows := make([][]string, 0)
	for _, op := range ops {
		for _, t := range op.Transactions {
			if t.Trade == nil {
				continue
			}

			name, symbol, aType := "unknown", "unknown", "unknown"
			if a, ok := assets[t.AssetID]; ok {
				name, symbol, aType = a.Name, a.Symbol, a.Type
			}

			if assetType != "" && aType != assetType {
				continue
			}

			rows = append(rows, []string{
				t.CreditedAt,
				op.OperationType,
				name,
				symbol,
				aType,
				moneyValue(t.AssetAmount),
				t.Trade.Rate,
				t.Trade.ToEurRate,
				t.Trade.TradeID,
			})

			if limit > 0 && len(rows) >= limit {
				return output.Render(app.outFormat, columns, rows)
			}
		}
	}

	return output.Render(app.outFormat, columns, rows)
}
