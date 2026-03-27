package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// TableFormatter renders data as a human-readable table.
type TableFormatter struct{}

func (t *TableFormatter) Format(w io.Writer, columns []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, strings.Join(columns, "\t"))

	// Separator
	seps := make([]string, len(columns))
	for i, col := range columns {
		seps[i] = strings.Repeat("-", len(col))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Rows
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
}
