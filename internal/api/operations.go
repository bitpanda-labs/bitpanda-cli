package api

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
)

// Trade holds the trade details attached to a transaction (buy/sell).
type Trade struct {
	TradeID       string `json:"trade_id"`
	Fee           *Money `json:"fee"`
	FeePercentage string `json:"fee_percentage"`
	Rate          string `json:"rate"`
	RateWithFee   string `json:"rate_with_fee"`
	ToEurRate     string `json:"to_eur_rate"`
}

// CompensatesInfo describes the operation/transaction that a transaction compensates.
type CompensatesInfo struct {
	OperationType   string `json:"operation_type"`
	TransactionType string `json:"transaction_type"`
}

// OperationTransaction is a single transaction within an operation.
type OperationTransaction struct {
	TransactionID     string           `json:"transaction_id"`
	AssetID           string           `json:"asset_id"`
	CurrencyID        string           `json:"currency_id"`
	WalletID          string           `json:"wallet_id"`
	WalletOwner       string           `json:"wallet_owner"`
	AssetAmount       *Money           `json:"asset_amount"`
	FeeAmount         *Money           `json:"fee_amount"`
	TransactionType   string           `json:"transaction_type"`
	Flow              string           `json:"flow"`
	OrderID           string           `json:"order_id"`
	CreditedAt        string           `json:"credited_at"`
	AssetBalanceAfter *Money           `json:"asset_balance_after"`
	Compensates       string           `json:"compensates"`
	CompensatesInfo   *CompensatesInfo `json:"compensates_info"`
	IndexAssetID      string           `json:"index_asset_id"`
	Trade             *Trade           `json:"trade"`
}

// Operation is a single operation with its nested transactions.
type Operation struct {
	OperationID   string                 `json:"operation_id"`
	OperationType string                 `json:"operation_type"`
	Transactions  []OperationTransaction `json:"transactions"`
}

// OperationParams holds query parameters for listing operations.
type OperationParams struct {
	From        string
	To          string
	AssetIDs    []string
	CurrencyIDs []string
	PageSize    int
	Limit       int
	Progress    io.Writer
}

// ListOperations fetches operations with optional filtering, following pagination.
func (c *Client) ListOperations(ctx context.Context, p OperationParams) ([]Operation, error) {
	params := url.Values{}
	if p.From != "" {
		params.Set("from", p.From)
	}
	if p.To != "" {
		params.Set("to", p.To)
	}
	for _, id := range p.AssetIDs {
		params.Add("asset_id", id)
	}
	for _, id := range p.CurrencyIDs {
		params.Add("currency_id", id)
	}

	pageSize := p.PageSize
	if pageSize == 0 {
		pageSize = 25
	}

	rawItems, err := PaginateAll(ctx, c, "/v1/operations", params, pageSize, p.Limit, p.Progress)
	if err != nil {
		return nil, err
	}

	ops := make([]Operation, 0, len(rawItems))
	for _, raw := range rawItems {
		var op Operation
		if err := json.Unmarshal(raw, &op); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}
