package cli

import (
	"fmt"
	"strings"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerPortfolioHistory(parent *cobra.Command) {
	var (
		timeframe            string
		equivalentCurrencyID string
	)

	cmd := &cobra.Command{
		Use:   "portfolio-history",
		Short: "Show portfolio performance history over time",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPortfolioHistory(cmd, timeframe, equivalentCurrencyID)
		},
	}

	cmd.Flags().StringVar(&timeframe, "timeframe", "", "Timeframe: day, week, month, six_month, year")
	cmd.Flags().StringVar(&equivalentCurrencyID, "equivalent-currency-id", "", "Currency UUID for valuations")
	parent.AddCommand(cmd)
}

func (app *App) runPortfolioHistory(cmd *cobra.Command, timeframe, equivalentCurrencyID string) error {
	history, err := app.apiClient.GetPortfolioHistory(cmd.Context(), api.PortfolioHistoryParams{
		EquivalentCurrencyID: equivalentCurrencyID,
		Timeframe:            strings.ToUpper(timeframe),
	})
	if err != nil {
		return err
	}

	if history.ReturnPercentage != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Return over period: %s%%\n", history.ReturnPercentage)
	}

	columns := []string{"Time", "Value"}
	rows := make([][]string, 0, len(history.Datapoints))
	for _, dp := range history.Datapoints {
		rows = append(rows, []string{dp.Time, moneyValue(dp.Value)})
	}

	return output.Render(app.outFormat, columns, rows)
}
