package cli

import (
	"fmt"
	"strings"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerQuote(parent *cobra.Command) {
	quoteCmd := &cobra.Command{
		Use:   "quote",
		Short: "Create and accept trading quotes",
	}

	app.registerQuoteCreate(quoteCmd)
	app.registerQuoteAccept(quoteCmd)

	parent.AddCommand(quoteCmd)
}

func (app *App) registerQuoteCreate(parent *cobra.Command) {
	var (
		assetID    string
		currencyID string
		side       string
		quantity   string
		notional   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a trading quote (does not execute the trade)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runQuoteCreate(cmd, assetID, currencyID, side, quantity, notional)
		},
	}

	cmd.Flags().StringVar(&assetID, "asset-id", "", "Asset UUID (required)")
	cmd.Flags().StringVar(&currencyID, "currency-id", "", "Currency UUID (required)")
	cmd.Flags().StringVar(&side, "side", "", "buy or sell (required)")
	cmd.Flags().StringVar(&quantity, "quantity", "", "Asset amount (mutually exclusive with --notional)")
	cmd.Flags().StringVar(&notional, "notional", "", "Currency amount (mutually exclusive with --quantity)")
	_ = cmd.MarkFlagRequired("asset-id")
	_ = cmd.MarkFlagRequired("currency-id")
	_ = cmd.MarkFlagRequired("side")
	parent.AddCommand(cmd)
}

func (app *App) runQuoteCreate(cmd *cobra.Command, assetID, currencyID, side, quantity, notional string) error {
	sideUpper := strings.ToUpper(side)
	if sideUpper != "BUY" && sideUpper != "SELL" {
		return fmt.Errorf("invalid --side %q: must be buy or sell", side)
	}
	if (quantity == "") == (notional == "") {
		return fmt.Errorf("exactly one of --quantity or --notional must be provided")
	}

	resp, err := app.apiClient.CreateQuote(cmd.Context(), api.CreateQuoteRequest{
		AssetID:    assetID,
		CurrencyID: currencyID,
		Side:       sideUpper,
		Quantity:   quantity,
		Notional:   notional,
	})
	if err != nil {
		return err
	}

	columns := []string{"Quote ID", "Status", "Side", "Price", "Quantity", "Notional", "Expires At"}
	var price, qty, not, expires string
	if resp.Quote != nil {
		price = resp.Quote.Price
		qty = resp.Quote.Quantity
		not = resp.Quote.Notional
		expires = resp.Quote.ExpiresAt
	}
	rows := [][]string{{resp.QuoteID, resp.Status, resp.Side, price, qty, not, expires}}

	if app.outFormat == output.FormatTable {
		fmt.Fprintf(cmd.ErrOrStderr(), "Quote created. Accept it with: bp quote accept %s\n", resp.QuoteID)
	}
	return output.Render(app.outFormat, columns, rows)
}

func (app *App) registerQuoteAccept(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "accept <QUOTE_ID>",
		Short: "Accept a quote and execute the trade",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runQuoteAccept(cmd, args[0])
		},
	}

	parent.AddCommand(cmd)
}

func (app *App) runQuoteAccept(cmd *cobra.Command, quoteID string) error {
	prompt := fmt.Sprintf("Accept quote %s and execute the trade?", quoteID)
	if !app.assumeYes && !confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), prompt) {
		fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
		return nil
	}

	resp, err := app.apiClient.AcceptQuote(cmd.Context(), quoteID)
	if err != nil {
		return err
	}

	columns := []string{"Quote ID", "Trade ID", "Status", "Side", "Price", "Quantity", "Notional", "Executed At"}
	var price, qty, not, executed string
	if resp.Execution != nil {
		price = resp.Execution.Price
		qty = resp.Execution.Quantity
		not = resp.Execution.Notional
		executed = resp.Execution.ExecutedAt
	}
	rows := [][]string{{resp.QuoteID, resp.TradeID, resp.Status, resp.Side, price, qty, not, executed}}
	return output.Render(app.outFormat, columns, rows)
}
