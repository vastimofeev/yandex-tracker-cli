#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION=""
REPO="vasti/yandex-tracker-cli"
FROM_RELEASE="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --repo)
      REPO="$2"
      shift 2
      ;;
    --from-release)
      FROM_RELEASE="true"
      shift
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

mkdir -p "$INSTALL_DIR"
OUTPUT="$INSTALL_DIR/yt"

if [[ "$FROM_RELEASE" == "true" ]]; then
  OS="$(uname -s)"
  ARCH="$(uname -m)"
  case "$OS" in
    Linux) OS_NAME="linux" ;;
    Darwin) OS_NAME="darwin" ;;
    *)
      echo "Unsupported operating system: $OS" >&2
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64|amd64) ARCH_NAME="amd64" ;;
    arm64|aarch64) ARCH_NAME="arm64" ;;
    *)
      echo "Unsupported architecture: $ARCH" >&2
      exit 1
      ;;
  esac

  ASSET_NAME="yt-${OS_NAME}-${ARCH_NAME}"
  if [[ -n "$VERSION" ]]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"
  else
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"
  fi

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$OUTPUT"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$OUTPUT" "$DOWNLOAD_URL"
  else
    echo "curl or wget is required to download a release asset" >&2
    exit 1
  fi
  chmod +x "$OUTPUT"
else
  REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  "$REPO_ROOT/scripts/build.sh" "$OUTPUT"
fi

cat <<EOF
Installed yt to $OUTPUT
Make sure $INSTALL_DIR is in PATH.
EOF
"$OUTPUT" version
