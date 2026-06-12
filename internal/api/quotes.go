package api

import (
	"context"
)

// CreateQuoteRequest is the payload for creating a trading quote. Exactly one of
// Quantity (asset amount) or Notional (currency amount) must be set.
type CreateQuoteRequest struct {
	AssetID    string `json:"asset_id"`
	CurrencyID string `json:"currency_id"`
	Side       string `json:"side"` // BUY or SELL
	Quantity   string `json:"quantity,omitempty"`
	Notional   string `json:"notional,omitempty"`
}

// QuoteDetail holds the priced terms of a created quote.
type QuoteDetail struct {
	Price            string `json:"price"`
	PriceWithPremium string `json:"price_with_premium"`
	Premium          string `json:"premium"`
	Quantity         string `json:"quantity"`
	Notional         string `json:"notional"`
	ExpiresAt        string `json:"expires_at"`
	ToEuroRate       string `json:"to_euro_rate"`
}

// CreateQuoteResponse is the result of creating a quote.
type CreateQuoteResponse struct {
	QuoteID    string       `json:"quote_id"`
	Status     string       `json:"status"`
	Side       string       `json:"side"`
	AssetID    string       `json:"asset_id"`
	CurrencyID string       `json:"currency_id"`
	Quote      *QuoteDetail `json:"quote"`
}

// CreateQuote requests a tradable quote.
func (c *Client) CreateQuote(ctx context.Context, req CreateQuoteRequest) (*CreateQuoteResponse, error) {
	var resp struct {
		Data CreateQuoteResponse `json:"data"`
	}
	if err := c.PostJSON(ctx, "/v1/quotes", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// AcceptQuoteExecution holds the execution terms of an accepted quote.
type AcceptQuoteExecution struct {
	Price            string `json:"price"`
	PriceWithPremium string `json:"price_with_premium"`
	Premium          string `json:"premium"`
	Quantity         string `json:"quantity"`
	Notional         string `json:"notional"`
	ExecutedAt       string `json:"executed_at"`
	ToEuroRate       string `json:"to_euro_rate"`
}

// AcceptQuoteTax holds tax details for an accepted quote, when present.
type AcceptQuoteTax struct {
	TaxAmount              string `json:"tax_amount"`
	PessimisticWithholding *bool  `json:"pessimistic_withholding"`
}

// AcceptQuoteResponse is the result of accepting a quote.
type AcceptQuoteResponse struct {
	QuoteID    string                `json:"quote_id"`
	TradeID    string                `json:"trade_id"`
	Status     string                `json:"status"`
	Side       string                `json:"side"`
	AssetID    string                `json:"asset_id"`
	CurrencyID string                `json:"currency_id"`
	Execution  *AcceptQuoteExecution `json:"execution"`
	Tax        *AcceptQuoteTax       `json:"tax"`
}

// AcceptQuote accepts a previously created quote by ID, executing the trade.
func (c *Client) AcceptQuote(ctx context.Context, quoteID string) (*AcceptQuoteResponse, error) {
	var resp struct {
		Data AcceptQuoteResponse `json:"data"`
	}
	if err := c.PostJSON(ctx, "/v1/quotes/"+quoteID+"/accept", nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
