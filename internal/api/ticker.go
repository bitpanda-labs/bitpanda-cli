package api

import (
	"context"
	"net/url"
)

// Ticker represents the price of a single asset, as returned by
// GET /v1/tickers/{assetId}.
type Ticker struct {
	AssetID    string `json:"asset_id"`
	Price      string `json:"price"`
	CurrencyID string `json:"currency_id"`
}

// GetTicker fetches the ticker for a single asset by ID.
func (c *Client) GetTicker(ctx context.Context, assetID string) (*Ticker, error) {
	var resp struct {
		Data Ticker `json:"data"`
	}
	if err := c.GetJSON(ctx, "/v1/tickers/"+assetID, url.Values{}, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
