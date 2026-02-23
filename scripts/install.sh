#!/usr/bin/env sh
set -e
REPO="shravan20/vecna"
VERSION="${VERSION:-latest}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) echo "Unsupported OS"; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported arch"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  TAG=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
else
  TAG="$VERSION"
fi

URL="https://github.com/$REPO/releases/download/v${TAG}/vecna_${TAG}_${OS}_${ARCH}.tar.gz"
mkdir -p "$BIN_DIR"
curl -sL "$URL" | tar -xz -C "$BIN_DIR" vecna
chmod +x "$BIN_DIR/vecna"
echo "Installed vecna ${TAG} to $BIN_DIR"
