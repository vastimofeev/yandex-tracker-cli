# Yandex Tracker CLI

`yt` is a cross-platform Go CLI for the Yandex Tracker API.

It is designed for two usage modes:

- manual terminal work with concise text output
- agent and automation workflows with stable `--json` output

The CLI currently covers:

- authentication and session inspection
- issue read and write operations
- issue comments, worklog, links, and checklist items
- queue metadata and queue field inspection
- Tracker dictionaries such as fields, issue types, statuses, and priorities
- Tracker entities: projects, portfolios, and goals

## Install

### Windows

From the repository root:

```powershell
.\scripts\install.ps1
```

This installs `yt.exe` to `$HOME\.local\bin` and can add that directory to the user `PATH`.

### macOS and Linux

From the repository root:

```bash
./scripts/install.sh
```

This installs `yt` to `~/.local/bin`.

### Install from GitHub Release

Windows:

```powershell
.\scripts\install.ps1 -FromRelease
```

macOS or Linux:

```bash
./scripts/install.sh --from-release
```

You can also pin a specific tag:

```powershell
.\scripts\install.ps1 -FromRelease -Version v1.3.0
```

```bash
./scripts/install.sh --from-release --version v1.3.0
```

## Build

Run locally without installation:

```powershell
go run ./cmd/yandex-tracker-cli -- version
```

Build a local binary:

```powershell
go build -o .\yt.exe ./cmd/yandex-tracker-cli
```

## Authentication

The CLI stores the saved auth context in the system keyring.

Interactive login:

```powershell
yt auth login
```

Direct token login:

```powershell
yt --token <oauth-token> --org-id 7942431 auth login
```

Inspect the saved session:

```powershell
yt auth status
yt me
```

Clear the saved session:

```powershell
yt auth logout
```

## Output Modes

Text output is intended for quick terminal inspection.

For agents and automation, prefer:

```powershell
yt <command> --json
```

Examples:

```powershell
yt me --json
yt issue search --query "Queue: DV" --per-page 20 --json
yt entity list --type project --page 2 --per-page 20 --json
```

## Supported Commands

### Session and Identity

- `yt auth login`
- `yt auth status`
- `yt auth logout`
- `yt me`
- `yt version`

### Issues

- `yt issue get <issue-key>`
- `yt issue search`
- `yt issue count`
- `yt issue create`
- `yt issue edit <issue-key>`
- `yt issue transitions <issue-key>`
- `yt issue transition <issue-key>`

### Issue Comments

- `yt issue comment add <issue-key>`
- `yt issue comment list <issue-key>`
- `yt issue comment edit <issue-key> <comment-id>`

### Issue Worklog

- `yt issue worklog add <issue-key>`
- `yt issue worklog list <issue-key>`

### Issue Links

- `yt issue links list <issue-key>`
- `yt issue links add <issue-key>`

### Issue Checklist

- `yt issue checklist list <issue-key>`
- `yt issue checklist add <issue-key>`
- `yt issue checklist update <issue-key> <item-id>`
- `yt issue checklist check <issue-key> <item-id>`
- `yt issue checklist uncheck <issue-key> <item-id>`
- `yt issue checklist delete <issue-key> <item-id>`

### Queues and Metadata

- `yt queue list`
- `yt queue get <queue-key>`
- `yt queue fields <queue-key>`
- `yt queue local-fields <queue-key>`
- `yt field list`
- `yt issuetype list`
- `yt status list`
- `yt priority list`

### Entities

- `yt entity list --type project|portfolio|goal`
- `yt entity get <id> --type project|portfolio|goal`

## Common Examples

Create an issue:

```powershell
yt issue create --queue DV --summary "Smoke test" --description "Created from yt"
```

Search issues:

```powershell
yt issue search --query "Queue: DV AND Status: \"Open\"" --per-page 10 --json
```

Inspect queue fields:

```powershell
yt queue fields DV --json
```

List projects:

```powershell
yt entity list --type project --per-page 20 --page 2 --json
```

## Development

Recommended commands:

```powershell
go test ./...
go test -cover ./...
gofmt -w .
go vet ./...
```

If Go cannot write to the default build cache on your machine, use a workspace-local cache:

```powershell
New-Item -ItemType Directory -Force .gocache | Out-Null
$env:GOCACHE = (Resolve-Path .\.gocache).Path
go test ./...
```
