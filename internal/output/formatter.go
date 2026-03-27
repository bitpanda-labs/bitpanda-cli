// Package output provides a Formatter interface and implementations for
// rendering tabular data in table, JSON, and CSV formats.
package output

import (
	"fmt"
	"io"
	"os"
)

// Format represents an output format.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
)

// ParseFormat parses a string into a Format, returning an error for invalid values.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("invalid output format %q: must be table, json, or csv", s)
	}
}

// Formatter renders structured data to a writer.
type Formatter interface {
	Format(w io.Writer, columns []string, rows [][]string) error
}

// NewFormatter creates a formatter for the given format.
func NewFormatter(f Format) Formatter {
	switch f {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatCSV:
		return &CSVFormatter{}
	default:
		return &TableFormatter{}
	}
}

// Render is a convenience function that formats data to stdout.
func Render(f Format, columns []string, rows [][]string) error {
	return NewFormatter(f).Format(os.Stdout, columns, rows)
}
