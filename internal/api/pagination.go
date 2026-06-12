package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// PaginatedResponse represents a cursor-paginated API response. The public API
// returns the next page's cursor in next_cursor, with has_next_page signalling
// whether more pages remain.
type PaginatedResponse struct {
	Data        json.RawMessage `json:"data"`
	SelfCursor  string          `json:"self_cursor"`
	NextCursor  string          `json:"next_cursor"`
	HasNextPage bool            `json:"has_next_page"`
}

// GetNextCursor returns the cursor for the next page.
func (p *PaginatedResponse) GetNextCursor() string {
	return p.NextCursor
}

// PaginateAll fetches all pages from a cursor-paginated endpoint. The cursor is
// always passed via the "cursor" query parameter.
// limit of 0 means no limit. progress receives a dot per page if non-nil.
func PaginateAll(ctx context.Context, c *Client, path string, baseParams url.Values, pageSize int, limit int, progress io.Writer) ([]json.RawMessage, error) {
	const cursorParam = "cursor"
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
