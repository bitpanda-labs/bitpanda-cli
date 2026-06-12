package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Get_SetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Api-Key"); got != "test-key" {
			t.Errorf("X-Api-Key = %q, want %q", got, "test-key")
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("User-Agent header is empty")
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	c := NewClient("test-key", false)
	c.BaseURL = server.URL

	_, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_PostJSON_SendsBodyAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["foo"] != "bar" {
			t.Errorf("body foo = %q, want bar", body["foo"])
		}
		w.Write([]byte(`{"data":{"status":"OK"}}`))
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := c.PostJSON(context.Background(), "/test", map[string]string{"foo": "bar"}, &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data.Status != "OK" {
		t.Errorf("status = %q, want OK", resp.Data.Status)
	}
}

func TestClient_Get_ReturnsAPIErrorOn401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`Unauthorized`))
	}))
	defer server.Close()

	c := NewClient("bad-key", false)
	c.BaseURL = server.URL

	_, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() to be true")
	}
}

func TestClient_Get_ReturnsAPIErrorOn500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	_, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", apiErr.StatusCode)
	}
}

func TestPaginateAll_MultiplePages(t *testing.T) {
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		cursor := r.URL.Query().Get("cursor")
		if page == 2 && cursor != "cursor-1" {
			t.Errorf("page 2: cursor = %q, want %q", cursor, "cursor-1")
		}
		var resp PaginatedResponse
		switch page {
		case 1:
			resp = PaginatedResponse{
				Data:        json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				NextCursor:  "cursor-1",
				HasNextPage: true,
			}
		case 2:
			resp = PaginatedResponse{
				Data:        json.RawMessage(`[{"id":"3"}]`),
				NextCursor:  "",
				HasNextPage: false,
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	items, err := PaginateAll(context.Background(), c, "/test", nil, 10, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestPaginateAll_RespectsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[{"id":"1"},{"id":"2"},{"id":"3"},{"id":"4"},{"id":"5"}]`),
			HasNextPage: true,
			NextCursor:  "cursor",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	items, err := PaginateAll(context.Background(), c, "/test", nil, 10, 3, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestSanitizeBody_PlainText(t *testing.T) {
	got := sanitizeBody([]byte("Internal Server Error"))
	if got != "Internal Server Error" {
		t.Errorf("sanitizeBody = %q, want %q", got, "Internal Server Error")
	}
}

func TestSanitizeBody_StripsHTML(t *testing.T) {
	got := sanitizeBody([]byte("<html><body><h1>Error</h1><p>Something went wrong</p></body></html>"))
	if got != "ErrorSomething went wrong" {
		t.Errorf("sanitizeBody = %q, want %q", got, "ErrorSomething went wrong")
	}
}

func TestSanitizeBody_ExtractsErrorCode(t *testing.T) {
	got := sanitizeBody([]byte(`{"error":{"code":"insufficient_funds"}}`))
	if got != "insufficient_funds" {
		t.Errorf("sanitizeBody = %q, want %q", got, "insufficient_funds")
	}
}

func TestSanitizeBody_Truncates(t *testing.T) {
	long := strings.Repeat("x", 300)
	got := sanitizeBody([]byte(long))
	if len(got) != 203 { // 200 + "..."
		t.Errorf("sanitizeBody length = %d, want 203", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("expected truncated body to end with '...'")
	}
}

func TestGetNextCursor(t *testing.T) {
	tests := []struct {
		name       string
		nextCursor string
		want       string
	}{
		{"has next", "next-1", "next-1"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaginatedResponse{NextCursor: tt.nextCursor}
			if got := p.GetNextCursor(); got != tt.want {
				t.Errorf("GetNextCursor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListAssets_SendsParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("symbol"); got != "BTC" {
			t.Errorf("symbol = %q, want BTC", got)
		}
		if got := q.Get("page_size"); got != "50" {
			t.Errorf("page_size = %q, want 50", got)
		}
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[{"id":"a1","name":"Bitcoin","symbol":"BTC","type":"cryptocoin"}]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	assets, err := c.ListAssets(context.Background(), AssetParams{Symbol: "BTC", PageSize: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assets) != 1 || assets[0].Symbol != "BTC" {
		t.Fatalf("unexpected assets: %+v", assets)
	}
}

func TestGetAsset_FoundAndNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") == "a1" {
			resp := PaginatedResponse{Data: json.RawMessage(`[{"id":"a1","name":"Bitcoin","symbol":"BTC"}]`)}
			json.NewEncoder(w).Encode(resp)
			return
		}
		json.NewEncoder(w).Encode(PaginatedResponse{Data: json.RawMessage(`[]`)})
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	asset, err := c.GetAsset(context.Background(), "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset == nil || asset.Name != "Bitcoin" {
		t.Fatalf("expected Bitcoin, got %+v", asset)
	}

	missing, err := c.GetAsset(context.Background(), "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing asset, got %+v", missing)
	}
}

func TestGetTicker_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tickers/a1" {
			t.Errorf("path = %q, want /v1/tickers/a1", r.URL.Path)
		}
		w.Write([]byte(`{"data":{"asset_id":"a1","price":"50000","currency_id":"eur"}}`))
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	ticker, err := c.GetTicker(context.Background(), "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticker.Price != "50000" {
		t.Errorf("price = %q, want 50000", ticker.Price)
	}
}

func TestListOperations_SendsParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("from"); got != "2024-01-01T00:00:00Z" {
			t.Errorf("from = %q", got)
		}
		ids := q["asset_id"]
		if len(ids) != 2 {
			t.Errorf("asset_id count = %d, want 2", len(ids))
		}
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[{"operation_id":"op1","operation_type":"buy","transactions":[{"transaction_id":"tx1","asset_id":"a1","trade":{"trade_id":"t1"}}]}]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	ops, err := c.ListOperations(context.Background(), OperationParams{
		From:     "2024-01-01T00:00:00Z",
		AssetIDs: []string{"a1", "a2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) != 1 || ops[0].Transactions[0].Trade.TradeID != "t1" {
		t.Fatalf("unexpected operations: %+v", ops)
	}
}

func TestListCurrencies_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"c1","symbol":"EUR","name":"Euro"}]}`))
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	currencies, err := c.ListCurrencies(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(currencies) != 1 || currencies[0].Symbol != "EUR" {
		t.Fatalf("unexpected currencies: %+v", currencies)
	}
}

func TestGetPortfolio_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"asset_id":"a1","balance":{"value":"2.0","asset_id":"a1"},"currency_balance":{"value":"100000","currency_id":"eur"}}]}`))
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	positions, err := c.GetPortfolio(context.Background(), PortfolioParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(positions) != 1 || positions[0].Balance.Value != "2.0" {
		t.Fatalf("unexpected positions: %+v", positions)
	}
}

func TestCreateAndAcceptQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/quotes":
			w.Write([]byte(`{"data":{"quote_id":"q1","status":"OPEN","side":"BUY","quote":{"price":"50000"}}}`))
		case "/v1/quotes/q1/accept":
			w.Write([]byte(`{"data":{"quote_id":"q1","trade_id":"t1","status":"FINISHED","execution":{"price":"50000"}}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	created, err := c.CreateQuote(context.Background(), CreateQuoteRequest{AssetID: "a1", CurrencyID: "eur", Side: "BUY", Notional: "10"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if created.QuoteID != "q1" {
		t.Errorf("quote_id = %q, want q1", created.QuoteID)
	}

	accepted, err := c.AcceptQuote(context.Background(), "q1")
	if err != nil {
		t.Fatalf("accept error: %v", err)
	}
	if accepted.TradeID != "t1" {
		t.Errorf("trade_id = %q, want t1", accepted.TradeID)
	}
}

func TestExecuteEarnAction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/earn/actions/STAKE" {
			t.Errorf("path = %q, want /v1/earn/actions/STAKE", r.URL.Path)
		}
		var body map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		if string(body["asset_amount"]) != "1.5" {
			t.Errorf("asset_amount = %s, want 1.5 (as number)", body["asset_amount"])
		}
		w.Write([]byte(`{"data":{"status":"INITIATED"}}`))
	}))
	defer server.Close()

	c := NewClient("key", false)
	c.BaseURL = server.URL

	result, err := c.ExecuteEarnAction(context.Background(), "STAKE", EarnActionRequest{ConfigID: "cfg1", AssetAmount: "1.5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "INITIATED" {
		t.Errorf("status = %q, want INITIATED", result.Status)
	}
}
