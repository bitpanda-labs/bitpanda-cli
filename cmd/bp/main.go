// Package main is the entrypoint for the bp CLI tool.
package main

import (
	"errors"
	"os"

	"github.com/bitpanda-labs/bitpanda-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
