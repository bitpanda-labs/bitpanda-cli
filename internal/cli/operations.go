package cli

import (
	"io"
	"os"
	"strings"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func (app *App) registerOperations(parent *cobra.Command) {
	var (
		assetIDs    []string
		currencyIDs []string
		from        string
		to          string
		limit       int
		pageSize    int
		all         bool
	)

	cmd := &cobra.Command{
		Use:   "operations",
		Short: "List account operations and their transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runOperations(cmd, assetIDs, currencyIDs, from, to, limit, pageSize, all)
		},
	}

	cmd.Flags().StringSliceVar(&assetIDs, "asset-id", nil, "Filter by asset UUID (repeatable)")
	cmd.Flags().StringSliceVar(&currencyIDs, "currency-id", nil, "Filter by currency UUID (repeatable)")
	cmd.Flags().StringVar(&from, "from", "", "From date (ISO 8601 date-time)")
	cmd.Flags().StringVar(&to, "to", "", "To date (ISO 8601 date-time)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of operations")
	cmd.Flags().IntVar(&pageSize, "page-size", 25, "Operations per API page (1-100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (may be slow)")
	parent.AddCommand(cmd)
}

func (app *App) runOperations(cmd *cobra.Command, assetIDs, currencyIDs []string, from, to string, limit, pageSize int, all bool) error {
	fetchLimit := limit
	if !all && fetchLimit == 0 {
		fetchLimit = pageSize
	}

	var progress io.Writer
	if all {
		if f, ok := cmd.ErrOrStderr().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			progress = f
		}
	}

	ops, err := app.apiClient.ListOperations(cmd.Context(), api.OperationParams{
		AssetIDs:    assetIDs,
		CurrencyIDs: currencyIDs,
		From:        from,
		To:          to,
		PageSize:    pageSize,
		Limit:       fetchLimit,
		Progress:    progress,
	})
	if err != nil {
		return err
	}

	columns := []string{"Operation ID", "Operation", "Transaction ID", "Type", "Flow", "Asset/Currency", "Amount", "Fee", "Date", "Trade ID"}
	rows := make([][]string, 0, len(ops))
	for _, op := range ops {
		if len(op.Transactions) == 0 {
			rows = append(rows, []string{op.OperationID, op.OperationType, "", "", "", "", "", "", "", ""})
			continue
		}
		for _, t := range op.Transactions {
			assetOrCurrency := t.AssetID
			if assetOrCurrency == "" {
				assetOrCurrency = t.CurrencyID
			}
			tradeID := ""
			if t.Trade != nil {
				tradeID = t.Trade.TradeID
			}
			rows = append(rows, []string{
				op.OperationID,
				op.OperationType,
				t.TransactionID,
				t.TransactionType,
				strings.ToLower(t.Flow),
				assetOrCurrency,
				moneyValue(t.AssetAmount),
				moneyValue(t.FeeAmount),
				t.CreditedAt,
				tradeID,
			})
		}
	}

	return output.Render(app.outFormat, columns, rows)
}
