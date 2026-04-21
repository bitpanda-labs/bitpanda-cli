package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// FlexInt is an int that can be unmarshaled from both JSON numbers and strings.
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	// Try number first
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*fi = FlexInt(n)
		return nil
	}
	// Try string
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("FlexInt: cannot parse %q as int: %w", s, err)
		}
		*fi = FlexInt(n)
		return nil
	}
	return fmt.Errorf("FlexInt: cannot unmarshal %s", string(b))
}

// PaginatedResponse represents a paginated API response with cursor fields.
// Supports both wallets/transactions shape (end_cursor) and ticker shape (next_cursor).
type PaginatedResponse struct {
	Data            json.RawMessage `json:"data"`
	StartCursor     string          `json:"start_cursor"`
	EndCursor       string          `json:"end_cursor"`
	NextCursor      string          `json:"next_cursor"`
	HasNextPage     bool            `json:"has_next_page"`
	HasPreviousPage bool            `json:"has_previous_page"`
	PageSize        FlexInt         `json:"page_size"`
}

// GetNextCursor returns the cursor for the next page,
// handling both response shapes.
func (p *PaginatedResponse) GetNextCursor() string {
	if p.NextCursor != "" {
		return p.NextCursor
	}
	return p.EndCursor
}

// PaginateAll fetches all pages from a paginated endpoint.
// cursorParam is the query parameter name for the cursor ("after" for wallets/transactions, "cursor" for ticker).
// limit of 0 means no limit. progress receives a dot per page if non-nil.
func PaginateAll(ctx context.Context, c *Client, path string, baseParams url.Values, cursorParam string, pageSize int, limit int, progress io.Writer) ([]json.RawMessage, error) {
	var allItems []json.RawMessage
	cursor := ""
	pagesWritten := 0

	for {
		params := url.Values{}
		for k, v := range baseParams {
			params[k] = v
		}
		if pageSize > 0 {
			params.Set("page_size", strconv.Itoa(pageSize))
		}
		if cursor != "" {
			params.Set(cursorParam, cursor)
		}

		var resp PaginatedResponse
		if err := c.GetJSON(ctx, path, params, &resp); err != nil {
			if pagesWritten > 0 && progress != nil {
				fmt.Fprintln(progress)
			}
			return nil, err
		}

		if progress != nil {
			fmt.Fprint(progress, ".")
			pagesWritten++
		}

		var items []json.RawMessage
		if err := json.Unmarshal(resp.Data, &items); err != nil {
			if pagesWritten > 0 && progress != nil {
				fmt.Fprintln(progress)
			}
			return nil, err
		}

		allItems = append(allItems, items...)

		if limit > 0 && len(allItems) >= limit {
			allItems = allItems[:limit]
			break
		}

		if !resp.HasNextPage {
			break
		}

		cursor = resp.GetNextCursor()
		if cursor == "" {
			break
		}
	}

	if pagesWritten > 0 && progress != nil {
		fmt.Fprintln(progress)
	}

	return allItems, nil
}
