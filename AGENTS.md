# Repository Guidelines

## Project Structure & Module Organization
This repository contains a cross-platform Go CLI for the Yandex Tracker API. Keep the layout close to standard Go conventions:

- `cmd/yandex-tracker-cli/` for the executable entry point
- `internal/` for private application packages such as auth, API client, services, output, and command handlers
- `pkg/` only for reusable public packages if they are truly needed
- `testdata/` for fixtures used by tests
- `scripts/` for install, update, and build helpers
- `.github/workflows/` for CI and release automation

Keep command wiring thin in `cmd/...`; put business logic in `internal/...`.

## Build, Test, and Development Commands
Use Go’s built-in toolchain first. Recommended commands:

- `go run ./cmd/yandex-tracker-cli` to run the CLI locally
- `go build ./cmd/yandex-tracker-cli` to build the binary
- `go build -o ./yt.exe ./cmd/yandex-tracker-cli` to produce a local test binary on Windows
- `go test ./...` to run all unit tests
- `go test -cover ./...` to check coverage
- `gofmt -w .` to format code
- `go vet ./...` to catch common mistakes
- `.\scripts\install.ps1` or `./scripts/install.sh` to install the CLI locally

If a `Makefile` is added later, it should wrap these commands instead of replacing them.

## Coding Style & Naming Conventions
Always format code with `gofmt`. Follow idiomatic Go:

- tabs for indentation, as produced by `gofmt`
- short package names in lowercase, for example `tracker`, `config`, `output`
- exported identifiers in `PascalCase`, internal helpers in `camelCase`
- files named by responsibility, for example `client.go`, `issues.go`, `auth_env.go`

Design the CLI to behave consistently on Windows, macOS, and Linux. Avoid shell-specific assumptions in paths, quoting, or environment handling.
Prefer stable JSON output for agent-facing workflows and concise text output for manual inspection.

## Testing Guidelines
Place tests next to the code they cover in `*_test.go` files. Prefer table-driven tests for command parsing, API request building, and output formatting. Mock Yandex Tracker HTTP interactions instead of calling the live service in unit tests.

Each change should include tests for the main success path and relevant error handling.
If a change affects `--help`, examples, or output formatting, update or add tests where it is practical.

## Commit & Pull Request Guidelines
There is no established Git history yet, so start with clear conventions:

- use imperative commit subjects, for example `Add issue list command`
- keep each commit focused on one logical change
- mention Tracker issues in the PR description when applicable

Pull requests should include a short summary, local test results, and notes about platform-specific behavior if it was changed.
If install, update, or release behavior changes, mention the exact commands used for validation.

## Security & Configuration Tips
Never commit OAuth tokens, IAM tokens, org IDs, or real Tracker credentials. Keep local secrets in environment variables or the system keyring and provide sanitized examples in documentation.
