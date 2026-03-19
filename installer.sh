#!/usr/bin/env bash
# installer.sh — One-command installer for git-user
set -e

# Configuration
REPO_URL="https://github.com/divyo-argha/git-user" # Update this to your real repo URL
BINARY="git-user"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "==> Setting up git-user installer..."

# Check dependencies
check_dep() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ Error: $1 is not installed. Please install it and try again."
    exit 1
  fi
}

check_dep "git"
check_dep "go"

# Create a temporary directory for building
TMP_DIR=$(mktemp -d -t git-user-build-XXXXXXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> Cloning repository..."
git clone --depth 1 "$REPO_URL" "$TMP_DIR"
cd "$TMP_DIR"

echo "==> Building $BINARY..."
go build -ldflags="-s -w" -o "$BINARY" .

echo "==> Installing to $INSTALL_DIR/$BINARY..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
else
  echo "ℹ Need sudo permissions to install to $INSTALL_DIR"
  sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
  sudo chmod +x "$INSTALL_DIR/$BINARY"
fi

echo ""
echo "✅ git-user installed successfully to $INSTALL_DIR/$BINARY"
echo ""
echo "Quick Start:"
echo "  git user add work  work@company.com"
echo "  git user add home  me@gmail.com"
echo "  git user switch work"
echo ""
echo "Run 'git user --help' for more information."
