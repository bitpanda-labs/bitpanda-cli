package api

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
)

// Transaction represents a transaction from the API.
type Transaction struct {
	TransactionID string `json:"transaction_id"`
	OperationID   string `json:"operation_id"`
	AssetID       string `json:"asset_id"`
	AccountID     string `json:"account_id"`
	WalletID      string `json:"wallet_id"`
	AssetAmount   string `json:"asset_amount"`
	FeeAmount     string `json:"fee_amount"`
	OperationType string `json:"operation_type"`
	TransType     string `json:"transaction_type"`
	Flow          string `json:"flow"`
	CreditedAt    string `json:"credited_at"`
	Compensates   string `json:"compensates"`
	TradeID       string `json:"trade_id"`
}

// TransactionParams holds query parameters for listing transactions.
type TransactionParams struct {
	WalletID string
	Flow     string
	AssetID  string
	From     string
	To       string
	PageSize int
	Limit    int
	Progress io.Writer
}

// ListTransactions fetches transactions with optional filtering.
func (c *Client) ListTransactions(ctx context.Context, p TransactionParams) ([]Transaction, error) {
	params := url.Values{}
	if p.WalletID != "" {
		params.Set("wallet_id", p.WalletID)
	}
	if p.Flow != "" {
		params.Set("flow", p.Flow)
	}
	if p.AssetID != "" {
		params.Set("asset_id", p.AssetID)
	}
	if p.From != "" {
		params.Set("from_including", p.From)
	}
	if p.To != "" {
		params.Set("to_excluding", p.To)
	}

	pageSize := p.PageSize
	if pageSize == 0 {
		pageSize = 25
	}

	rawItems, err := PaginateAll(ctx, c, "/v1/transactions", params, "after", pageSize, p.Limit, p.Progress)
	if err != nil {
		return nil, err
	}

	txns := make([]Transaction, 0, len(rawItems))
	for _, raw := range rawItems {
		var t Transaction
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, nil
}
