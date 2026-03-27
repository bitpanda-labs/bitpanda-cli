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

	c := NewClient("test-key")
	c.BaseURL = server.URL

	_, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Get_ReturnsAPIErrorOn401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`Unauthorized`))
	}))
	defer server.Close()

	c := NewClient("bad-key")
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

	c := NewClient("key")
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
		var resp PaginatedResponse
		switch page {
		case 1:
			resp = PaginatedResponse{
				Data:        json.RawMessage(`[{"id":"1"},{"id":"2"}]`),
				EndCursor:   "cursor-1",
				HasNextPage: true,
			}
		case 2:
			resp = PaginatedResponse{
				Data:        json.RawMessage(`[{"id":"3"}]`),
				EndCursor:   "",
				HasNextPage: false,
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	items, err := PaginateAll(context.Background(), c, "/test", nil, "after", 10, 0)
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
			EndCursor:   "cursor",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	items, err := PaginateAll(context.Background(), c, "/test", nil, "after", 10, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestPaginateAll_TickerCursorParam(t *testing.T) {
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		cursor := r.URL.Query().Get("cursor")
		if page == 2 && cursor != "next-1" {
			t.Errorf("page 2: cursor = %q, want %q", cursor, "next-1")
		}

		var resp PaginatedResponse
		switch page {
		case 1:
			resp = PaginatedResponse{
				Data:        json.RawMessage(`[{"symbol":"BTC"}]`),
				NextCursor:  "next-1",
				HasNextPage: true,
			}
		case 2:
			resp = PaginatedResponse{
				Data:        json.RawMessage(`[{"symbol":"ETH"}]`),
				HasNextPage: false,
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	items, err := PaginateAll(context.Background(), c, "/v1/ticker", nil, "cursor", 500, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}

func TestSanitizeBody_PlainText(t *testing.T) {
	got := sanitizeBody("Internal Server Error")
	if got != "Internal Server Error" {
		t.Errorf("sanitizeBody = %q, want %q", got, "Internal Server Error")
	}
}

func TestSanitizeBody_StripsHTML(t *testing.T) {
	got := sanitizeBody("<html><body><h1>Error</h1><p>Something went wrong</p></body></html>")
	if got != "ErrorSomething went wrong" {
		t.Errorf("sanitizeBody = %q, want %q", got, "ErrorSomething went wrong")
	}
}

func TestSanitizeBody_Truncates(t *testing.T) {
	long := strings.Repeat("x", 300)
	got := sanitizeBody(long)
	if len(got) != 203 { // 200 + "..."
		t.Errorf("sanitizeBody length = %d, want 203", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("expected truncated body to end with '...'")
	}
}

func TestFlexInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"number", `42`, 42, false},
		{"string number", `"25"`, 25, false},
		{"zero", `0`, 0, false},
		{"negative number", `-5`, -5, false},
		{"string negative", `"-5"`, -5, false},
		{"invalid string", `"abc"`, 0, true},
		{"boolean", `true`, 0, true},
		{"null", `null`, 0, false}, // json.Unmarshal accepts null for int (sets to zero value)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fi FlexInt
			err := fi.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && int(fi) != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %d, want %d", tt.input, int(fi), tt.want)
			}
		})
	}
}

func TestGetNextCursor(t *testing.T) {
	tests := []struct {
		name       string
		nextCursor string
		endCursor  string
		want       string
	}{
		{"next_cursor preferred", "next-1", "end-1", "next-1"},
		{"end_cursor fallback", "", "end-1", "end-1"},
		{"both empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaginatedResponse{
				NextCursor: tt.nextCursor,
				EndCursor:  tt.endCursor,
			}
			got := p.GetNextCursor()
			if got != tt.want {
				t.Errorf("GetNextCursor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListWallets_SendsParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("asset_id"); got != "asset-123" {
			t.Errorf("asset_id = %q, want %q", got, "asset-123")
		}
		if got := q.Get("page_size"); got != "50" {
			t.Errorf("page_size = %q, want %q", got, "50")
		}
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[{"wallet_id":"w1","asset_id":"asset-123","wallet_type":"","balance":"10.0","last_credited_at":"2024-01-01"}]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	wallets, err := c.ListWallets(context.Background(), WalletParams{
		AssetID:  "asset-123",
		PageSize: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wallets) != 1 {
		t.Fatalf("got %d wallets, want 1", len(wallets))
	}
	if wallets[0].WalletID != "w1" {
		t.Errorf("wallet_id = %q, want %q", wallets[0].WalletID, "w1")
	}
	if wallets[0].Balance != "10.0" {
		t.Errorf("balance = %q, want %q", wallets[0].Balance, "10.0")
	}
}

func TestListWallets_DefaultPageSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page_size"); got != "25" {
			t.Errorf("default page_size = %q, want %q", got, "25")
		}
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	_, err := c.ListWallets(context.Background(), WalletParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTransactions_SendsParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("wallet_id"); got != "w1" {
			t.Errorf("wallet_id = %q, want %q", got, "w1")
		}
		if got := q.Get("flow"); got != "incoming" {
			t.Errorf("flow = %q, want %q", got, "incoming")
		}
		if got := q.Get("asset_id"); got != "a1" {
			t.Errorf("asset_id = %q, want %q", got, "a1")
		}
		if got := q.Get("from_including"); got != "2024-01-01" {
			t.Errorf("from_including = %q, want %q", got, "2024-01-01")
		}
		if got := q.Get("to_excluding"); got != "2024-02-01" {
			t.Errorf("to_excluding = %q, want %q", got, "2024-02-01")
		}
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[{"transaction_id":"tx1","asset_id":"a1","operation_type":"buy","flow":"incoming","asset_amount":"1.0","fee_amount":"0.01","credited_at":"2024-01-15","trade_id":"t1"}]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	txns, err := c.ListTransactions(context.Background(), TransactionParams{
		WalletID: "w1",
		Flow:     "incoming",
		AssetID:  "a1",
		From:     "2024-01-01",
		To:       "2024-02-01",
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("got %d transactions, want 1", len(txns))
	}
	if txns[0].TransactionID != "tx1" {
		t.Errorf("transaction_id = %q, want %q", txns[0].TransactionID, "tx1")
	}
	if txns[0].FeeAmount != "0.01" {
		t.Errorf("fee_amount = %q, want %q", txns[0].FeeAmount, "0.01")
	}
}

func TestListTransactions_OmitsEmptyParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Has("wallet_id") {
			t.Error("wallet_id should not be sent when empty")
		}
		if q.Has("flow") {
			t.Error("flow should not be sent when empty")
		}
		if q.Has("asset_id") {
			t.Error("asset_id should not be sent when empty")
		}
		if q.Has("from_including") {
			t.Error("from_including should not be sent when empty")
		}
		if q.Has("to_excluding") {
			t.Error("to_excluding should not be sent when empty")
		}
		resp := PaginatedResponse{
			Data:        json.RawMessage(`[]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	_, err := c.ListTransactions(context.Background(), TransactionParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAsset_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/assets/a1" {
			t.Errorf("path = %q, want /v1/assets/a1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Asset{
			Data: AssetData{ID: "a1", Name: "Bitcoin", Symbol: "BTC"},
		})
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	asset, err := c.GetAsset(context.Background(), "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset.Data.ID != "a1" {
		t.Errorf("ID = %q, want %q", asset.Data.ID, "a1")
	}
	if asset.Data.Name != "Bitcoin" {
		t.Errorf("Name = %q, want %q", asset.Data.Name, "Bitcoin")
	}
	if asset.Data.Symbol != "BTC" {
		t.Errorf("Symbol = %q, want %q", asset.Data.Symbol, "BTC")
	}
}

func TestGetAsset_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`Not Found`))
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	_, err := c.GetAsset(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", apiErr.StatusCode)
	}
}

func TestFetchAllTicker_KeyedBySymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := PaginatedResponse{
			Data: json.RawMessage(`[
				{"id":"1","symbol":"BTC","type":"cryptocoin","currency":"EUR","price":"50000","price_change_day":"-2.5"},
				{"id":"2","symbol":"ETH","type":"cryptocoin","currency":"EUR","price":"3000","price_change_day":"1.2"}
			]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	ticker, err := c.FetchAllTicker(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ticker) != 2 {
		t.Fatalf("got %d entries, want 2", len(ticker))
	}
	btc, ok := ticker["BTC"]
	if !ok {
		t.Fatal("BTC not found in ticker map")
	}
	if btc.Price != "50000" {
		t.Errorf("BTC price = %q, want %q", btc.Price, "50000")
	}
	if btc.PriceChangeDay != "-2.5" {
		t.Errorf("BTC price_change_day = %q, want %q", btc.PriceChangeDay, "-2.5")
	}
}

func TestListAllAssets_KeyedByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := PaginatedResponse{
			Data: json.RawMessage(`[
				{"id":"a1","name":"Bitcoin","symbol":"BTC"},
				{"id":"a2","name":"Ethereum","symbol":"ETH"}
			]`),
			HasNextPage: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("key")
	c.BaseURL = server.URL

	assets, err := c.ListAllAssets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("got %d assets, want 2", len(assets))
	}
	btc, ok := assets["a1"]
	if !ok {
		t.Fatal("a1 not found in assets map")
	}
	if btc.Symbol != "BTC" {
		t.Errorf("a1 symbol = %q, want %q", btc.Symbol, "BTC")
	}
	if btc.Name != "Bitcoin" {
		t.Errorf("a1 name = %q, want %q", btc.Name, "Bitcoin")
	}
}
