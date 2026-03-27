package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

var (
	testCols = []string{"Name", "Value"}
	testRows = [][]string{
		{"Alice", "100"},
		{"Bob", "200"},
	}
)

func TestTableFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	if err := f.Format(&buf, testCols, testRows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Name") || !strings.Contains(out, "Value") {
		t.Errorf("table output missing headers: %s", out)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Bob") {
		t.Errorf("table output missing data: %s", out)
	}
}

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}
	if err := f.Format(&buf, testCols, testRows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[0]["Name"] != "Alice" {
		t.Errorf("expected Alice, got %s", result[0]["Name"])
	}
	if result[1]["Value"] != "200" {
		t.Errorf("expected 200, got %s", result[1]["Value"])
	}
}

func TestCSVFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &CSVFormatter{}
	if err := f.Format(&buf, testCols, testRows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}
	if lines[0] != "Name,Value" {
		t.Errorf("expected header 'Name,Value', got %q", lines[0])
	}
	if lines[1] != "Alice,100" {
		t.Errorf("expected 'Alice,100', got %q", lines[1])
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"table", FormatTable, false},
		{"json", FormatJSON, false},
		{"csv", FormatCSV, false},
		{"", FormatTable, false},
		{"xml", "", true},
	}

	for _, tt := range tests {
		f, err := ParseFormat(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if f != tt.want {
			t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, f, tt.want)
		}
	}
}
