package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerPortfolio(parent *cobra.Command) {
	var (
		sortFlag             string
		assetID              string
		currencyID           string
		equivalentCurrencyID string
	)

	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Show portfolio holdings with valuations and returns",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPortfolio(cmd, sortFlag, assetID, currencyID, equivalentCurrencyID)
		},
	}

	cmd.Flags().StringVar(&sortFlag, "sort", "value", "Sort by: name, value")
	cmd.Flags().StringVar(&assetID, "asset-id", "", "Filter by asset UUID")
	cmd.Flags().StringVar(&currencyID, "currency-id", "", "Filter by currency UUID")
	cmd.Flags().StringVar(&equivalentCurrencyID, "equivalent-currency-id", "", "Currency UUID for valuations")
	parent.AddCommand(cmd)
}

func (app *App) runPortfolio(cmd *cobra.Command, sortFlag, assetID, currencyID, equivalentCurrencyID string) error {
	ctx := cmd.Context()

	positions, err := app.apiClient.GetPortfolio(ctx, api.PortfolioParams{
		EquivalentCurrencyID: equivalentCurrencyID,
		CurrencyID:           currencyID,
		AssetID:              assetID,
	})
	if err != nil {
		return err
	}

	if len(positions) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No portfolio positions found.")
		return nil
	}

	// Enrich with symbols/names from assets and currencies.
	assets, err := app.apiClient.ListAllAssets(ctx)
	if err != nil {
		return fmt.Errorf("fetching assets: %w", err)
	}
	currencyList, err := app.apiClient.ListCurrencies(ctx, "")
	if err != nil {
		return fmt.Errorf("fetching currencies: %w", err)
	}
	currencies := make(map[string]api.Currency, len(currencyList))
	for _, c := range currencyList {
		currencies[c.ID] = c
	}

	type row struct {
		symbol    string
		name      string
		balance   string
		available string
		value     string
		valueNum  float64
		invested  string
		ret       string
		retPct    string
	}

	rows := make([]row, 0, len(positions))
	var total float64
	for _, p := range positions {
		symbol, name := assetLabel(p.AssetID, p.CurrencyID, assets, currencies)
		value := moneyValue(p.CurrencyBalance)
		// Fiat positions have no separate currency balance; use the balance itself.
		if value == "" {
			value = moneyValue(p.Balance)
		}
		valueNum, _ := strconv.ParseFloat(value, 64)
		total += valueNum

		rows = append(rows, row{
			symbol:    symbol,
			name:      name,
			balance:   moneyValue(p.Balance),
			available: moneyValue(p.AvailableBalance),
			value:     value,
			valueNum:  valueNum,
			invested:  moneyValue(p.InvestedAmount),
			ret:       moneyValue(p.TotalReturn),
			retPct:    p.TotalReturnPercent,
		})
	}

	switch sortFlag {
	case "name":
		sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	default: // value
		sort.Slice(rows, func(i, j int) bool { return rows[i].valueNum > rows[j].valueNum })
	}

	// "Available" is the balance free to trade; "Balance" includes staked or
	// otherwise locked funds, which cannot be traded.
	columns := []string{"Asset", "Name", "Balance", "Available", "Value", "Invested", "Return", "Return %"}
	tableRows := make([][]string, 0, len(rows)+1)
	for _, r := range rows {
		tableRows = append(tableRows, []string{
			r.symbol, r.name, r.balance, r.available, r.value, r.invested, r.ret, r.retPct,
		})
	}
	// The TOTAL summary row is a human-readable footer; appending it to JSON or
	// CSV output would surface a phantom holding to programmatic consumers, so
	// it is only rendered in table format.
	if app.outFormat == output.FormatTable {
		tableRows = append(tableRows, []string{"TOTAL", "", "", "", strconv.FormatFloat(total, 'f', 2, 64), "", "", ""})
	}

	return output.Render(app.outFormat, columns, tableRows)
}
