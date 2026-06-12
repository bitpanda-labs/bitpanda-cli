package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// Money mirrors the API's MoneyResponse: a string value with optional asset and
// currency identifiers.
type Money struct {
	Value      string `json:"value"`
	AssetID    string `json:"asset_id"`
	CurrencyID string `json:"currency_id"`
}

// Asset represents an asset from the /v1/assets endpoint.
type Asset struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	ISIN   string `json:"isin"`
	Group  string `json:"group"`
	Type   string `json:"type"`
}

// AssetParams holds query parameters for listing assets.
type AssetParams struct {
	ISIN     string
	Type     string
	Group    string
	Symbol   string
	ID       string
	PageSize int
	Limit    int
}

// ListAssets fetches assets with optional filtering, following pagination.
func (c *Client) ListAssets(ctx context.Context, p AssetParams) ([]Asset, error) {
	params := url.Values{}
	if p.ISIN != "" {
		params.Set("isin", p.ISIN)
	}
	if p.Type != "" {
		params.Set("type", p.Type)
	}
	if p.Group != "" {
		params.Set("group", p.Group)
	}
	if p.Symbol != "" {
		params.Set("symbol", p.Symbol)
	}
	if p.ID != "" {
		params.Set("id", p.ID)
	}

	pageSize := p.PageSize
	if pageSize == 0 {
		pageSize = 25
	}

	rawItems, err := PaginateAll(ctx, c, "/v1/assets", params, pageSize, p.Limit, nil)
	if err != nil {
		return nil, err
	}

	assets := make([]Asset, 0, len(rawItems))
	for _, raw := range rawItems {
		var a Asset
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// GetAsset fetches a single asset by ID via the id filter on /v1/assets.
// Returns nil if no asset matches.
func (c *Client) GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	assets, err := c.ListAssets(ctx, AssetParams{ID: assetID, PageSize: 1})
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		return nil, nil
	}
	return &assets[0], nil
}

// ListAllAssets fetches all assets via pagination and returns a map keyed by
// asset ID, used to enrich other responses with names and symbols.
func (c *Client) ListAllAssets(ctx context.Context) (map[string]Asset, error) {
	assets, err := c.ListAssets(ctx, AssetParams{PageSize: 100})
	if err != nil {
		return nil, err
	}
	result := make(map[string]Asset, len(assets))
	for _, a := range assets {
		result[a.ID] = a
	}
	return result, nil
}
