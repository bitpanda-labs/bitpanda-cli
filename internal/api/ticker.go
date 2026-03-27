package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// TickerEntry represents a single ticker item.
type TickerEntry struct {
	ID             string `json:"id"`
	Symbol         string `json:"symbol"`
	Type           string `json:"type"`
	Currency       string `json:"currency"`
	Price          string `json:"price"`
	PriceChangeDay string `json:"price_change_day"`
}

// FetchAllTicker fetches all ticker entries with auto-pagination.
// Returns a map keyed by symbol.
func (c *Client) FetchAllTicker(ctx context.Context) (map[string]TickerEntry, error) {
	rawItems, err := PaginateAll(ctx, c, "/v1/ticker", url.Values{}, "cursor", 500, 0)
	if err != nil {
		return nil, err
	}

	result := make(map[string]TickerEntry, len(rawItems))
	for _, raw := range rawItems {
		var entry TickerEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		result[entry.Symbol] = entry
	}
	return result, nil
}
