package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func (app *App) registerPrice(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "price <SYMBOL>",
		Short: "Get current price for a single asset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runPrice(cmd, args)
		},
	}

	parent.AddCommand(cmd)
}

func (app *App) runPrice(cmd *cobra.Command, args []string) error {
	symbol := strings.ToUpper(args[0])

	// NOTE: FetchAllTicker paginates through all entries to find a single symbol.
	// The Bitpanda API does not expose a single-symbol ticker endpoint.
	// If performance is a concern, consider caching the ticker map.
	ticker, err := app.apiClient.FetchAllTicker(cmd.Context())
	if err != nil {
		return err
	}

	entry, found := ticker.BySymbol[symbol]
	if !found {
		return fmt.Errorf("symbol %q not found", symbol)
	}

	columns := []string{"Symbol", "Price", "Currency", "24h Change"}
	rows := [][]string{
		{entry.Symbol, entry.Price, entry.Currency, entry.PriceChangeDay + "%"},
	}

	return output.Render(app.outFormat, columns, rows)
}
