package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func (app *App) registerTrades(parent *cobra.Command) {
	var (
		operation string
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
			return app.runTrades(cmd, operation, assetType, from, to, limit, all)
		},
	}

	cmd.Flags().StringVar(&operation, "operation", "", "Filter: buy, sell")
	cmd.Flags().StringVar(&assetType, "asset-type", "", "Filter: cryptocoin, metal, stock, etf, commodity")
	cmd.Flags().StringVar(&from, "from", "", "From date (ISO 8601, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "To date (ISO 8601, exclusive)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of trades (0 = all)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (may be slow with many trades)")
	parent.AddCommand(cmd)
}

func (app *App) runTrades(cmd *cobra.Command, operation, assetType, from, to string, limit int, all bool) error {
	ctx := cmd.Context()

	// Default to fetching one page (100 transactions) unless --all or --limit is set.
	fetchLimit := limit
	if !all && fetchLimit == 0 {
		fetchLimit = 100
	}

	// Fetch more transactions than the limit when asset-type filtering is needed
	if assetType != "" && fetchLimit > 0 {
		fetchLimit = fetchLimit * 10
	}

	txns, err := app.apiClient.ListTransactions(ctx, api.TransactionParams{
		From:     from,
		To:       to,
		PageSize: 100,
		Limit:    fetchLimit,
	})
	if err != nil {
		return err
	}

	// Filter for trades (have trade_id, incoming, buy/sell)
	var trades []api.Transaction
	for _, t := range txns {
		if t.TradeID == "" {
			continue
		}
		if t.Flow != "incoming" {
			continue
		}
		if operation != "" && t.OperationType != operation {
			continue
		}
		if operation == "" && t.OperationType != "buy" && t.OperationType != "sell" {
			continue
		}
		trades = append(trades, t)
	}

	// Fetch ticker — provides name, symbol, type, and price keyed by asset ID
	ticker, err := app.apiClient.FetchAllTicker(ctx)
	if err != nil {
		return fmt.Errorf("fetching prices: %w", err)
	}

	// Build enriched rows
	type enrichedTrade struct {
		Date      string
		Operation string
		Name      string
		Symbol    string
		AssetType string
		Amount    string
		EURPrice  string
		TradeID   string
	}

	var enriched []enrichedTrade
	for _, t := range trades {
		name := "unknown"
		symbol := "unknown"
		aType := "unknown"
		eurPrice := "N/A"
		if te, found := ticker.ByID[t.AssetID]; found {
			name = te.Name
			symbol = te.Symbol
			aType = te.Type
			eurPrice = te.Price
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: asset %s not found in ticker\n", t.AssetID)
		}

		if assetType != "" && aType != assetType {
			continue
		}

		enriched = append(enriched, enrichedTrade{
			Date:      t.CreditedAt,
			Operation: t.OperationType,
			Name:      name,
			Symbol:    symbol,
			AssetType: aType,
			Amount:    t.AssetAmount,
			EURPrice:  eurPrice,
			TradeID:   t.TradeID,
		})
	}

	// Apply limit
	if limit > 0 && len(enriched) > limit {
		enriched = enriched[:limit]
	}

	columns := []string{"Date", "Operation", "Asset", "Symbol", "Type", "Amount", "EUR Price", "Trade ID"}
	rows := make([][]string, 0, len(enriched))
	for _, e := range enriched {
		price := e.EURPrice
		if p, err := strconv.ParseFloat(price, 64); err == nil {
			price = formatFloat(p)
		}
		rows = append(rows, []string{
			e.Date,
			e.Operation,
			e.Name,
			e.Symbol,
			e.AssetType,
			e.Amount,
			price,
			e.TradeID,
		})
	}

	return output.Render(app.outFormat, columns, rows)
}
