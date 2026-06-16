# bitpanda-cli (`bp`)

[![CI](https://github.com/bitpanda-labs/bitpanda-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/bitpanda-labs/bitpanda-cli/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bitpanda-labs/bitpanda-cli)](https://github.com/bitpanda-labs/bitpanda-cli/blob/main/go.mod)
[![License](https://img.shields.io/github/license/bitpanda-labs/bitpanda-cli)](https://github.com/bitpanda-labs/bitpanda-cli/blob/main/LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/bitpanda-labs/bitpanda-cli)](https://github.com/bitpanda-labs/bitpanda-cli/releases/latest)

A command-line tool for the [Bitpanda Public API](https://docs.public.bitpanda.com/). View your portfolio, check prices, browse trades and operations, and place trades — all from your terminal.

## Installation

### Homebrew (macOS / Linux)

```bash
brew install bitpanda-labs/tap/bp
```

### Shell script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/bitpanda-labs/bitpanda-cli/main/install.sh | sh
```

### Debian / Ubuntu (.deb)

Download the `.deb` package from the [latest release](https://github.com/bitpanda-labs/bitpanda-cli/releases/latest) and install:

```bash
sudo dpkg -i bp_*_linux_amd64.deb
```

### Fedora / RHEL (.rpm)

Download the `.rpm` package from the [latest release](https://github.com/bitpanda-labs/bitpanda-cli/releases/latest) and install:

```bash
sudo rpm -i bp_*_linux_amd64.rpm
```

### Go

```bash
go install github.com/bitpanda-labs/bitpanda-cli/cmd/bp@latest
```

### From source

```bash
git clone https://github.com/bitpanda-labs/bitpanda-cli.git
cd bitpanda-cli
make install
```

### Pre-built binaries

Download from [Releases](https://github.com/bitpanda-labs/bitpanda-cli/releases/latest).

## Configuration

`bp` needs a Bitpanda API key. Get one at [app.bitpanda.com/my-account/apikey](https://app.bitpanda.com/my-account/apikey).

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

> **Warning:** The `--api-key` flag is the least secure option — command-line arguments are visible in process listings (`ps`, `/proc`) and may be recorded in your shell history. Prefer the `BITPANDA_API_KEY` environment variable or the config file.

> **Tip:** `bp` will warn on stderr if the config file has permissions more permissive than `0600`. Since the file contains your API key, restrict access with `chmod 600 ~/.config/bitpanda/config.yaml`.

## Usage

### Portfolio

```bash
bp portfolio                 # holdings with valuations and returns
bp portfolio --sort name     # sort by asset name (default: value, descending)
bp portfolio --asset-id UUID # single position
bp portfolio -o json         # JSON output
```

Each position shows total balance, available balance, value, invested amount,
and total return. **Balance** includes staked or otherwise locked funds;
**Available** is the portion free to trade. Pass `--equivalent-currency-id UUID`
to value the portfolio in a specific currency.

### Portfolio history

```bash
bp portfolio-history                   # performance over time
bp portfolio-history --timeframe month # day, week, month, six_month, year
```

### Operations

Operations are the canonical record of account activity. Each operation groups
one or more transactions (e.g. a buy debits fiat and credits the asset).

```bash
bp operations                       # recent operations (first page)
bp operations --all                 # full history (may be slow)
bp operations --asset-id UUID       # filter by asset (repeatable)
bp operations --currency-id UUID    # filter by currency (repeatable)
bp operations --from 2024-01-01T00:00:00Z --to 2024-12-31T00:00:00Z
bp operations -o csv                # CSV output for spreadsheets
```

### Trades

```bash
bp trades                          # recent trades (first page)
bp trades --all                    # full trade history (may be slow)
bp trades --asset-type cryptocoin  # only crypto trades
bp trades --from 2024-01-01T00:00:00Z --to 2024-06-30T00:00:00Z
bp trades --limit 20
```

### Prices

The ticker API is per-asset. `bp price` accepts a symbol (resolved via the
assets endpoint) or an asset UUID directly.

```bash
bp price BTC       # single asset price by symbol
bp price btc       # case-insensitive
bp price UUID      # by asset UUID
bp prices          # prices for held assets
```

### Assets & currencies

```bash
bp asset UUID                 # asset metadata by UUID
bp assets                     # list available assets
bp assets --type cryptocoin   # filter by type, group, symbol, or isin
bp currencies                 # list available fiat currencies
```

### Earn

```bash
bp earn configs                                       # list earn configurations
bp earn stake --config-id UUID --amount 1.5           # stake an asset
bp earn unstake --config-id UUID --amount 1.5         # unstake an asset
```

> **Note:** `earn stake` and `earn unstake` move funds and prompt for confirmation. Pass `-y`/`--yes` to skip the prompt (e.g. in scripts).

### Quotes (trading)

Trading is a two-step flow: create a quote to see the price, then accept it to
execute the trade.

```bash
# Create a quote (does not execute)
bp quote create --asset-id UUID --currency-id UUID --side buy --notional 100
bp quote create --asset-id UUID --currency-id UUID --side sell --quantity 0.5

# Accept a quote to execute the trade
bp quote accept QUOTE_ID
```

> **Note:** `quote accept` executes a real trade and prompts for confirmation. Pass `-y`/`--yes` to skip the prompt. Provide exactly one of `--notional` (currency amount) or `--quantity` (asset amount) when creating a quote.

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

Apache 2.0 — see [LICENSE](LICENSE) for details.
