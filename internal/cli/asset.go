package cli

import (
	"github.com/spf13/cobra"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
)

func (app *App) registerAsset(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "asset <ID>",
		Short: "Look up asset metadata by UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runAsset(cmd, args)
		},
	}

	parent.AddCommand(cmd)
}

func (app *App) runAsset(cmd *cobra.Command, args []string) error {
	assetID := args[0]

	asset, err := app.apiClient.GetAsset(cmd.Context(), assetID)
	if err != nil {
		return err
	}

	columns := []string{"ID", "Name", "Symbol"}
	rows := [][]string{
		{asset.Data.ID, asset.Data.Name, asset.Data.Symbol},
	}

	return output.Render(app.outFormat, columns, rows)
}
