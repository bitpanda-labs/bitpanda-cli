package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// EarnConfig represents an earn asset configuration.
type EarnConfig struct {
	ID                   string  `json:"id"`
	AssetID              string  `json:"asset_id"`
	AnnualPercentageRate float64 `json:"annual_percentage_rate"`
	Type                 string  `json:"type"`
	Mode                 string  `json:"mode"`
	Enabled              bool    `json:"enabled"`
	Soldout              bool    `json:"soldout"`
}

// EarnConfigParams holds query parameters for listing earn configs.
type EarnConfigParams struct {
	AssetID  string
	Type     string
	Mode     string
	PageSize int
	Limit    int
}

// ListEarnConfigs fetches earn asset configurations, following pagination.
func (c *Client) ListEarnConfigs(ctx context.Context, p EarnConfigParams) ([]EarnConfig, error) {
	params := url.Values{}
	if p.AssetID != "" {
		params.Set("assetId", p.AssetID)
	}
	if p.Type != "" {
		params.Set("type", p.Type)
	}
	if p.Mode != "" {
		params.Set("mode", p.Mode)
	}

	pageSize := p.PageSize
	if pageSize == 0 {
		pageSize = 20
	}

	rawItems, err := PaginateAll(ctx, c, "/v1/earn/configs", params, pageSize, p.Limit, nil)
	if err != nil {
		return nil, err
	}

	configs := make([]EarnConfig, 0, len(rawItems))
	for _, raw := range rawItems {
		var cfg EarnConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// EarnActionRequest is the payload for staking or unstaking.
type EarnActionRequest struct {
	ConfigID    string `json:"config_id"`
	WalletID    string `json:"wallet_id,omitempty"`
	AssetAmount string `json:"asset_amount"`
}

// EarnActionResult is the result of an earn action.
type EarnActionResult struct {
	Status string `json:"status"`
}

// ExecuteEarnAction stakes or unstakes (action must be "STAKE" or "UNSTAKE").
func (c *Client) ExecuteEarnAction(ctx context.Context, action string, req EarnActionRequest) (*EarnActionResult, error) {
	// asset_amount is typed as a JSON number upstream; marshal it as such while
	// keeping the CLI-facing value a string. json.Number validates the literal
	// and preserves full precision (no float64 round-trip).
	payload := map[string]any{
		"config_id":    req.ConfigID,
		"asset_amount": json.Number(req.AssetAmount),
	}
	if req.WalletID != "" {
		payload["wallet_id"] = req.WalletID
	}

	var resp struct {
		Data EarnActionResult `json:"data"`
	}
	if err := c.PostJSON(ctx, "/v1/earn/actions/"+action, payload, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
