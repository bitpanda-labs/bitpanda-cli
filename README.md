# bitpanda-cli (`bp`)

[![CI](https://github.com/bitpanda-labs/bitpanda-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/bitpanda-labs/bitpanda-cli/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bitpanda-labs/bitpanda-cli)](https://github.com/bitpanda-labs/bitpanda-cli/blob/main/go.mod)
[![License](https://img.shields.io/github/license/bitpanda-labs/bitpanda-cli)](https://github.com/bitpanda-labs/bitpanda-cli/blob/main/LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/bitpanda-labs/bitpanda-cli)](https://github.com/bitpanda-labs/bitpanda-cli/releases/latest)

A command-line tool for the [Bitpanda Developer API](https://developers.bitpanda.com/platform/). View your portfolio, check prices, browse trades and transactions — all from your terminal.

## Installation

### Homebrew

```bash
brew install bitpanda-labs/tap/bp
```

### From source

```bash
git clone https://github.com/bitpanda-labs/bitpanda-cli.git
cd bitpanda-cli
make install
```

### Pre-built binaries

Download from [Releases](https://github.com/bitpanda-labs/bitpanda-cli/releases).

## Configuration

`bp` needs a Bitpanda API key. Get one at [bitpanda.com/my-account/apikey](https://web.bitpanda.com/my-account/apikey).

Three ways to provide it (in priority order):

```bash
# 1. Flag (highest priority)
bp portfolio --api-key YOUR_KEY

# 2. Environment variable
export BITPANDA_API_KEY=YOUR_KEY
bp portfolio

# 3. Config file (~/.config/bitpanda/config.yaml)
mkdir -p ~/.config/bitpanda
echo "api_key: YOUR_KEY" > ~/.config/bitpanda/config.yaml
chmod 600 ~/.config/bitpanda/config.yaml
bp portfolio
```

> **Tip:** `bp` will warn on stderr if the config file has permissions more permissive than `0600`. Since the file contains your API key, restrict access with `chmod 600 ~/.config/bitpanda/config.yaml`.

## Usage

### Portfolio

```bash
bp portfolio              # aggregated holdings with EUR valuations
bp portfolio --sort value # sort by EUR value (descending)
bp portfolio -o json      # JSON output
```

> **Note:** The `portfolio` command uses the Bitpanda ticker API, which returns all prices in EUR. All valuations shown (EUR Price, EUR Value, TOTAL) assume EUR as the base currency. The `price` command displays the raw `Currency` field from the API response, which currently also returns EUR but is shown explicitly per asset.

### Balances

```bash
bp balances                  # all wallets
bp balances --non-zero       # only wallets with balance > 0
bp balances --asset-id UUID  # filter by asset
bp balances --limit 10       # cap results
```

### Trades

```bash
bp trades                          # recent buy/sell trades
bp trades --operation buy          # only buys
bp trades --asset-type cryptocoin  # only crypto trades
bp trades --from 2024-01-01 --to 2024-06-30
bp trades --limit 20
```

### Transactions

```bash
bp transactions                       # all transactions
bp transactions --flow incoming       # only incoming
bp transactions --wallet-id UUID
bp transactions --from 2024-01-01 --to 2024-12-31
bp transactions -o csv                # CSV output for spreadsheets
```

### Prices

```bash
bp price BTC       # single asset price
bp price btc       # case-insensitive
bp prices          # prices for held assets
bp prices --all    # all available prices
```

### Asset lookup

```bash
bp asset UUID      # asset metadata by ID
```

### Shell Completion

```bash
bp completion bash       # generate bash completions
bp completion zsh        # generate zsh completions
bp completion fish       # generate fish completions
bp completion powershell # generate PowerShell completions
```

To load completions in your current shell session:

```bash
# Bash
source <(bp completion bash)

# Zsh
source <(bp completion zsh)

# Fish
bp completion fish | source

# PowerShell
bp completion powershell | Out-String | Invoke-Expression
```

## Output Formats

All commands support `-o`/`--output` with three formats:

| Format | Flag | Description |
|--------|------|-------------|
| Table | `--output table` (default) | Human-readable table |
| JSON | `--output json` | For scripting and piping |
| CSV | `--output csv` | For spreadsheets |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication error |
| 3 | API error |

## Development

```bash
make build    # build to bin/bp
make test     # run tests
make install  # install to $GOPATH/bin
make lint     # run linter
```

## License

See [LICENSE](LICENSE) for details.
