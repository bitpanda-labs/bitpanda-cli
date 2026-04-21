package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// TickerEntry represents a single ticker item.
type TickerEntry struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	Type           string `json:"type"`
	Currency       string `json:"currency"`
	Price          string `json:"price"`
	PriceChangeDay string `json:"price_change_day"`
}

// Ticker holds ticker entries indexed by symbol and by asset ID.
type Ticker struct {
	BySymbol map[string]TickerEntry
	ByID     map[string]TickerEntry
}

// FetchAllTicker fetches all ticker entries with auto-pagination.
// Returns a Ticker with maps keyed by symbol and by asset ID.
func (c *Client) FetchAllTicker(ctx context.Context) (*Ticker, error) {
	rawItems, err := PaginateAll(ctx, c, "/v1/ticker", url.Values{}, "cursor", 500, 0, nil)
	if err != nil {
		return nil, err
	}

	t := &Ticker{
		BySymbol: make(map[string]TickerEntry, len(rawItems)),
		ByID:     make(map[string]TickerEntry, len(rawItems)),
	}
	for _, raw := range rawItems {
		var entry TickerEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		t.BySymbol[entry.Symbol] = entry
		t.ByID[entry.ID] = entry
	}
	return t, nil
}
