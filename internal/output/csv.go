package output

import (
	"encoding/csv"
	"io"
)

// CSVFormatter renders data as CSV with a header row.
type CSVFormatter struct{}

func (c *CSVFormatter) Format(w io.Writer, columns []string, rows [][]string) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(columns); err != nil {
		return err
	}
	for _, row := range rows {
		sanitized := make([]string, len(row))
		for i, cell := range row {
			sanitized[i] = sanitizeCSVField(cell)
		}
		if err := writer.Write(sanitized); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// sanitizeCSVField neutralizes spreadsheet formula injection. A cell whose
// first character is one of = + - @ (or the control lead-ins tab / carriage
// return) would be executed as a formula when the CSV is opened in Excel,
// Google Sheets, or LibreOffice. Per OWASP guidance, prefix such cells with a
// single quote so they are treated as literal text.
func sanitizeCSVField(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}
