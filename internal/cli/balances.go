package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func (app *App) registerBalances(parent *cobra.Command) {
	var (
		assetID  string
		nonZero  bool
		limit    int
		pageSize int
	)

	cmd := &cobra.Command{
		Use:   "balances",
		Short: "List wallet balances",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runBalances(cmd, assetID, nonZero, limit, pageSize)
		},
	}

	cmd.Flags().StringVar(&assetID, "asset-id", "", "Filter by asset UUID")
	cmd.Flags().BoolVar(&nonZero, "non-zero", false, "Only show wallets with balance > 0")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results (0 = all)")
	cmd.Flags().IntVar(&pageSize, "page-size", 25, "Items per API page (1-100)")
	parent.AddCommand(cmd)
}

func (app *App) runBalances(cmd *cobra.Command, assetID string, nonZero bool, limit, pageSize int) error {
	wallets, err := app.apiClient.ListWallets(cmd.Context(), api.WalletParams{
		AssetID:  assetID,
		PageSize: pageSize,
		Limit:    limit,
	})
	if err != nil {
		return err
	}

	if nonZero {
		var filtered []api.Wallet
		for _, w := range wallets {
			bal, err := strconv.ParseFloat(w.Balance, 64)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: skipping wallet %s: invalid balance %q: %v\n", w.WalletID, w.Balance, err)
				continue
			}
			if bal > 0 {
				filtered = append(filtered, w)
			}
		}
		wallets = filtered
	}

	columns := []string{"Wallet ID", "Asset ID", "Wallet Type", "Balance", "Last Credited"}
	rows := make([][]string, 0, len(wallets))
	for _, w := range wallets {
		wType := w.WalletType
		if wType == "" {
			wType = "regular"
		}
		rows = append(rows, []string{
			w.WalletID,
			w.AssetID,
			wType,
			w.Balance,
			w.LastCreditedAt,
		})
	}

	return output.Render(app.outFormat, columns, rows)
}
