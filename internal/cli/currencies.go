package cli

import (
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerCurrencies(parent *cobra.Command) {
	var id string

	cmd := &cobra.Command{
		Use:   "currencies",
		Short: "List available fiat currencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runCurrencies(cmd, id)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Filter by currency UUID")
	parent.AddCommand(cmd)
}

func (app *App) runCurrencies(cmd *cobra.Command, id string) error {
	currencies, err := app.apiClient.ListCurrencies(cmd.Context(), id)
	if err != nil {
		return err
	}

	columns := []string{"ID", "Symbol", "Name"}
	rows := make([][]string, 0, len(currencies))
	for _, c := range currencies {
		rows = append(rows, []string{c.ID, c.Symbol, c.Name})
	}

	return output.Render(app.outFormat, columns, rows)
}
