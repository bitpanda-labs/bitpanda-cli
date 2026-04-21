package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// Wallet represents a wallet from the API.
type Wallet struct {
	WalletID       string `json:"wallet_id"`
	AssetID        string `json:"asset_id"`
	WalletType     string `json:"wallet_type"`
	IndexAssetID   string `json:"index_asset_id"`
	LastCreditedAt string `json:"last_credited_at"`
	Balance        string `json:"balance"`
}

// WalletParams holds query parameters for listing wallets.
type WalletParams struct {
	AssetID  string
	PageSize int
	Limit    int
}

// ListWallets fetches all wallets with optional filtering.
func (c *Client) ListWallets(ctx context.Context, p WalletParams) ([]Wallet, error) {
	params := url.Values{}
	if p.AssetID != "" {
		params.Set("asset_id", p.AssetID)
	}

	pageSize := p.PageSize
	if pageSize == 0 {
		pageSize = 25
	}

	rawItems, err := PaginateAll(ctx, c, "/v1/wallets/", params, "after", pageSize, p.Limit, nil)
	if err != nil {
		return nil, err
	}

	wallets := make([]Wallet, 0, len(rawItems))
	for _, raw := range rawItems {
		var w Wallet
		if err := json.Unmarshal(raw, &w); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, nil
}
