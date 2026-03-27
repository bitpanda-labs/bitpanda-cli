package output

import (
	"encoding/json"
	"io"
)

// JSONFormatter renders data as JSON.
type JSONFormatter struct{}

func (j *JSONFormatter) Format(w io.Writer, columns []string, rows [][]string) error {
	result := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		obj := make(map[string]string, len(columns))
		for i, col := range columns {
			if i < len(row) {
				obj[col] = row[i]
			}
		}
		result = append(result, obj)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
