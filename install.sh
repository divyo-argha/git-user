#!/usr/bin/env bash
# install.sh — build and install git-user
set -e

BINARY="git-user"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "==> Building $BINARY ..."
go build -ldflags="-s -w" -o "$BINARY" .

echo "==> Installing to $INSTALL_DIR/$BINARY ..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
else
  sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
  sudo chmod +x "$INSTALL_DIR/$BINARY"
fi

echo ""
echo "✔ git-user installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Quick start:"
echo "  git user add work  work@company.com"
echo "  git user add home  me@gmail.com"
echo "  git user list"
echo "  git user switch work"
