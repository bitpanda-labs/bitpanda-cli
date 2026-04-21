package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// AssetData holds the fields for an asset.
type AssetData struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// Asset represents an asset from the API.
type Asset struct {
	Data AssetData `json:"data"`
}

// GetAsset fetches a single asset by ID.
func (c *Client) GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	var asset Asset
	if err := c.GetJSON(ctx, "/v1/assets/"+assetID, url.Values{}, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

// ListAllAssets fetches all assets via pagination and returns a map keyed by asset ID.
func (c *Client) ListAllAssets(ctx context.Context) (map[string]AssetData, error) {
	rawItems, err := PaginateAll(ctx, c, "/v1/assets", url.Values{}, "after", 100, 0, nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string]AssetData, len(rawItems))
	for _, raw := range rawItems {
		var a AssetData
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		result[a.ID] = a
	}
	return result, nil
}
