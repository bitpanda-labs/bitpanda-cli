package cli

import (
	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func (app *App) registerTransactions(parent *cobra.Command) {
	var (
		walletID string
		flow     string
		assetID  string
		from     string
		to       string
		limit    int
		pageSize int
		all      bool
	)

	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "List all transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runTransactions(cmd, walletID, flow, assetID, from, to, limit, pageSize, all)
		},
	}

	cmd.Flags().StringVar(&walletID, "wallet-id", "", "Filter by wallet UUID")
	cmd.Flags().StringVar(&flow, "flow", "", "Filter: incoming, outgoing")
	cmd.Flags().StringVar(&assetID, "asset-id", "", "Filter by asset UUID")
	cmd.Flags().StringVar(&from, "from", "", "From date (ISO 8601, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "To date (ISO 8601, exclusive)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().IntVar(&pageSize, "page-size", 25, "Items per API page (1-100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (may be slow with many transactions)")
	parent.AddCommand(cmd)
}

func (app *App) runTransactions(cmd *cobra.Command, walletID, flow, assetID, from, to string, limit, pageSize int, all bool) error {
	fetchLimit := limit
	if !all && fetchLimit == 0 {
		fetchLimit = pageSize
	}

	txns, err := app.apiClient.ListTransactions(cmd.Context(), api.TransactionParams{
		WalletID: walletID,
		Flow:     flow,
		AssetID:  assetID,
		From:     from,
		To:       to,
		PageSize: pageSize,
		Limit:    fetchLimit,
	})
	if err != nil {
		return err
	}

	columns := []string{"Transaction ID", "Asset ID", "Operation", "Flow", "Amount", "Fee", "Date", "Trade ID"}
	rows := make([][]string, 0, len(txns))
	for _, t := range txns {
		rows = append(rows, []string{
			t.TransactionID,
			t.AssetID,
			t.OperationType,
			t.Flow,
			t.AssetAmount,
			t.FeeAmount,
			t.CreditedAt,
			t.TradeID,
		})
	}

	return output.Render(app.outFormat, columns, rows)
}
