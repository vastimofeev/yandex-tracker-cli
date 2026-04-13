#!/usr/bin/env bash
set -euo pipefail

OUTPUT="${1:-./yandex-tracker-cli}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-}"
COMMIT="${COMMIT:-}"
BUILD_DATE="${BUILD_DATE:-}"

if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$REPO_ROOT" describe --tags --always 2>/dev/null || true)"
  VERSION="${VERSION:-dev}"
fi

if [[ -z "$COMMIT" ]]; then
  COMMIT="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || true)"
  COMMIT="${COMMIT:-unknown}"
fi

if [[ -z "$BUILD_DATE" ]]; then
  BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
fi

cd "$REPO_ROOT"
go build -ldflags "-X main.buildVersion=$VERSION -X main.buildCommit=$COMMIT -X main.buildDate=$BUILD_DATE" -o "$OUTPUT" ./cmd/yandex-tracker-cli
