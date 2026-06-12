package api

import (
	"context"
	"net/url"
)

// Currency represents a fiat currency from the /v1/currencies endpoint.
type Currency struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// ListCurrencies fetches the available fiat currencies, optionally filtered by ID.
func (c *Client) ListCurrencies(ctx context.Context, id string) ([]Currency, error) {
	params := url.Values{}
	if id != "" {
		params.Set("id", id)
	}

	var resp struct {
		Data []Currency `json:"data"`
	}
	if err := c.GetJSON(ctx, "/v1/currencies", params, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
