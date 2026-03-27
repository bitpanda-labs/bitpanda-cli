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
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
