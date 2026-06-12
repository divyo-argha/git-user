#!/bin/sh
set -e

# git-user curl installer

REPO="divyo-argha/git-user"
BIN_NAME="git-user"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)     PLATFORM="linux";;
    Darwin*)    PLATFORM="darwin";;
    *)          echo "Unsupported OS: $OS"; exit 1;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)     ARCHITECTURE="x86_64";;
    amd64)      ARCHITECTURE="x86_64";;
    aarch64)    ARCHITECTURE="arm64";;
    arm64)      ARCHITECTURE="arm64";;
    *)          echo "Unsupported architecture: $ARCH"; exit 1;;
esac

echo "Detected platform: $PLATFORM ($ARCHITECTURE)"

# Get the latest release URL
LATEST_URL=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep "browser_download_url.*${PLATFORM}_${ARCHITECTURE}.tar.gz" | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
    echo "Error: Could not find a release for $PLATFORM $ARCHITECTURE"
    exit 1
fi

echo "Downloading $BIN_NAME from $LATEST_URL ..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download and extract
curl -sL "$LATEST_URL" -o release.tar.gz
tar -xzf release.tar.gz "$BIN_NAME"

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Requires sudo privileges to install to $INSTALL_DIR."
    sudo mv "$BIN_NAME" "$INSTALL_DIR/"
else
    mv "$BIN_NAME" "$INSTALL_DIR/"
fi

# Make executable
sudo chmod +x "$INSTALL_DIR/$BIN_NAME" || chmod +x "$INSTALL_DIR/$BIN_NAME"

# Clean up
cd - > /dev/null
rm -rf "$TMP_DIR"

echo ""
echo "✅ Successfully installed $BIN_NAME to $INSTALL_DIR"
echo "Run 'git-user --help' to get started."
