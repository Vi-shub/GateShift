#!/usr/bin/env bash
# One-line install:
#   curl -fsSL https://raw.githubusercontent.com/vi-shub/gateshift/main/scripts/install.sh | bash
set -euo pipefail

REPO="${GATESHIFT_REPO:-vi-shub/gateshift}"
VERSION="${GATESHIFT_VERSION:-latest}"
INSTALL_DIR="${GATESHIFT_INSTALL_DIR:-${HOME}/bin}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
need curl
need tar
need uname

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac
case "$OS" in
  linux|darwin) ;;
  msys*|mingw*|cygwin*) OS=windows ;;
  *) echo "unsupported os: $OS" >&2; exit 1 ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
fi
VERSION="${VERSION#v}"
if [[ -z "$VERSION" ]]; then
  echo "could not resolve release version (publish a GitHub release first)" >&2
  exit 1
fi

EXT="tar.gz"
ASSET="gateshift_${VERSION}_${OS}_${ARCH}.${EXT}"
if [[ "$OS" == "windows" ]]; then
  EXT="zip"
  ASSET="gateshift_${VERSION}_${OS}_${ARCH}.${EXT}"
fi
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
echo "Downloading ${URL}"
curl -fsSL -o "${TMP}/${ASSET}" "$URL"

mkdir -p "$INSTALL_DIR"
if [[ "$EXT" == "zip" ]]; then
  need unzip
  unzip -qo "${TMP}/${ASSET}" -d "$TMP/out"
else
  mkdir -p "$TMP/out"
  tar -xzf "${TMP}/${ASSET}" -C "$TMP/out"
fi

BIN="$(find "$TMP/out" -type f -name 'gateshift*' | head -n1)"
if [[ -z "$BIN" ]]; then
  echo "gateshift binary not found in archive" >&2
  exit 1
fi
install -m 0755 "$BIN" "${INSTALL_DIR}/gateshift"
echo "Installed ${INSTALL_DIR}/gateshift (v${VERSION})"
echo "Ensure ${INSTALL_DIR} is on your PATH."
"${INSTALL_DIR}/gateshift" version
