package api

import (
	"context"
	"net/url"
)

// PortfolioPosition represents a single holding in the portfolio overview.
// assetId is set for non-fiat positions; currencyId is set for fiat positions.
type PortfolioPosition struct {
	AssetID            string `json:"asset_id"`
	CurrencyID         string `json:"currency_id"`
	Balance            *Money `json:"balance"`
	AvailableBalance   *Money `json:"available_balance"`
	InvestedAmount     *Money `json:"invested_amount"`
	AverageBuyPrice    *Money `json:"average_buy_price"`
	CurrencyBalance    *Money `json:"currency_balance"`
	TotalReturn        *Money `json:"total_return"`
	TotalReturnPercent string `json:"total_return_percent"`
}

// PortfolioParams holds query parameters for the portfolio overview.
type PortfolioParams struct {
	EquivalentCurrencyID string
	CurrencyID           string
	AssetID              string
}

// GetPortfolio fetches the portfolio overview for the authenticated user.
func (c *Client) GetPortfolio(ctx context.Context, p PortfolioParams) ([]PortfolioPosition, error) {
	params := url.Values{}
	if p.EquivalentCurrencyID != "" {
		params.Set("equivalent_currency_id", p.EquivalentCurrencyID)
	}
	if p.CurrencyID != "" {
		params.Set("currency_id", p.CurrencyID)
	}
	if p.AssetID != "" {
		params.Set("asset_id", p.AssetID)
	}

	var resp struct {
		Data []PortfolioPosition `json:"data"`
	}
	if err := c.GetJSON(ctx, "/v1/portfolio", params, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// PortfolioDatapoint is a single point in the portfolio performance history.
type PortfolioDatapoint struct {
	Time  string `json:"time"`
	Value *Money `json:"value"`
}

// PortfolioHistory holds the portfolio performance history.
type PortfolioHistory struct {
	Datapoints       []PortfolioDatapoint
	ReturnPercentage string
}

// PortfolioHistoryParams holds query parameters for the portfolio history.
type PortfolioHistoryParams struct {
	EquivalentCurrencyID string
	Timeframe            string // DAY, WEEK, MONTH, SIX_MONTH, YEAR
}

// GetPortfolioHistory fetches the portfolio performance history.
func (c *Client) GetPortfolioHistory(ctx context.Context, p PortfolioHistoryParams) (*PortfolioHistory, error) {
	params := url.Values{}
	if p.EquivalentCurrencyID != "" {
		params.Set("equivalent_currency_id", p.EquivalentCurrencyID)
	}
	if p.Timeframe != "" {
		params.Set("timeframe", p.Timeframe)
	}

	var resp struct {
		Data struct {
			Datapoints       []PortfolioDatapoint `json:"datapoints"`
			ReturnPercentage string               `json:"return_percentage"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, "/v1/portfolio-history", params, &resp); err != nil {
		return nil, err
	}
	return &PortfolioHistory{
		Datapoints:       resp.Data.Datapoints,
		ReturnPercentage: resp.Data.ReturnPercentage,
	}, nil
}
