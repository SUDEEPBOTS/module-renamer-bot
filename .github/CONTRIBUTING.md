# Contributing to SUDEEPBOTS Module Renamer

We welcome contributions to improve the refactoring engine, add new language parsers, and enhance performance!

## Development Guidelines

1. **Branching**: All development should be based on the `dev` branch.
2. **Code Style**: Format all Go code using `gofmt` and verify with `go vet ./...`.
3. **Tests**: Add unit tests for any new normalization tables, parser rules, or engines in `internal/renamer/engine_test.go`.
4. **Pull Requests**: Submit PRs targeting the `dev` branch with a clear description of changes.
