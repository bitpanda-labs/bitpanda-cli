package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/bitpanda-labs/bitpanda-cli/internal/api"
)

// moneyValue returns the string value of a Money pointer, or "" if nil.
func moneyValue(m *api.Money) string {
	if m == nil {
		return ""
	}
	return m.Value
}

// assetLabel resolves an asset or currency ID to a human-readable symbol using
// the provided lookup maps, falling back to the raw ID.
func assetLabel(assetID, currencyID string, assets map[string]api.Asset, currencies map[string]api.Currency) (symbol, name string) {
	if assetID != "" {
		if a, ok := assets[assetID]; ok {
			return a.Symbol, a.Name
		}
		return assetID, ""
	}
	if currencyID != "" {
		if cur, ok := currencies[currencyID]; ok {
			return cur.Symbol, cur.Name
		}
		return currencyID, ""
	}
	return "", ""
}

// confirm prompts the user on out and reads a yes/no answer from in. It returns
// true only if the user explicitly answers yes.
func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
