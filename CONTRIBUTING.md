# Contributing to bitpanda-cli

Thanks for your interest in contributing! Here's how to get started.

## Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [golangci-lint](https://golangci-lint.run/welcome/install-local/)

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint
```

## Pull Request Process

1. Fork the repository and create a feature branch from `main`
2. Make your changes, ensuring `make test` and `make lint` pass
3. Write clear commit messages describing the "why" not just the "what"
4. Open a pull request with a description of your changes

## Code Style

- Follow existing patterns in the codebase
- Run `make lint` before committing
- Keep changes focused — one concern per PR
