package cli

import (
	"fmt"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
	"github.com/bitpanda-labs/bitpanda-cli/internal/output"
	"github.com/spf13/cobra"
)

func (app *App) registerAsset(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "asset <UUID>",
		Short: "Look up asset metadata by UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runAsset(cmd, args)
		},
	}

	parent.AddCommand(cmd)
}

func (app *App) runAsset(cmd *cobra.Command, args []string) error {
	asset, err := app.apiClient.GetAsset(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if asset == nil {
		return fmt.Errorf("asset %q not found", args[0])
	}

	columns := []string{"ID", "Name", "Symbol", "ISIN", "Group", "Type"}
	rows := [][]string{
		{asset.ID, asset.Name, asset.Symbol, asset.ISIN, asset.Group, asset.Type},
	}

	return output.Render(app.outFormat, columns, rows)
}

func (app *App) registerAssets(parent *cobra.Command) {
	var (
		isin     string
		assetTyp string
		group    string
		symbol   string
		limit    int
		pageSize int
		all      bool
	)

	cmd := &cobra.Command{
		Use:   "assets",
		Short: "List available assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.runAssets(cmd, isin, assetTyp, group, symbol, limit, pageSize, all)
		},
	}

	cmd.Flags().StringVar(&isin, "isin", "", "Filter by ISIN")
	cmd.Flags().StringVar(&assetTyp, "type", "", "Filter by type")
	cmd.Flags().StringVar(&group, "group", "", "Filter by group")
	cmd.Flags().StringVar(&symbol, "symbol", "", "Filter by symbol")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().IntVar(&pageSize, "page-size", 25, "Items per API page (1-100)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages")
	parent.AddCommand(cmd)
}

func (app *App) runAssets(cmd *cobra.Command, isin, assetTyp, group, symbol string, limit, pageSize int, all bool) error {
	fetchLimit := limit
	if !all && fetchLimit == 0 {
		fetchLimit = pageSize
	}

	assets, err := app.apiClient.ListAssets(cmd.Context(), api.AssetParams{
		ISIN:     isin,
		Type:     assetTyp,
		Group:    group,
		Symbol:   symbol,
		PageSize: pageSize,
		Limit:    fetchLimit,
	})
	if err != nil {
		return err
	}

	columns := []string{"ID", "Name", "Symbol", "ISIN", "Group", "Type"}
	rows := make([][]string, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, []string{a.ID, a.Name, a.Symbol, a.ISIN, a.Group, a.Type})
	}

	return output.Render(app.outFormat, columns, rows)
}
