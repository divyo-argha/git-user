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
    MINGW*|MSYS*|CYGWIN*)
        echo "This installer supports macOS and Linux only."
        echo ""
        echo "On Windows, install git-user with npm instead:"
        echo "    npm install -g git-userhub"
        echo ""
        echo "The command will be available as 'git-user' after install."
        exit 1;;
    *)          echo "Error: Unsupported OS: $OS"; exit 1;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)     ARCHITECTURE="x86_64";;
    amd64)      ARCHITECTURE="x86_64";;
    aarch64)    ARCHITECTURE="arm64";;
    arm64)      ARCHITECTURE="arm64";;
    *)          echo "Error: Unsupported architecture: $ARCH"; exit 1;;
esac

echo "Detected platform: $PLATFORM ($ARCHITECTURE)"

# Get the latest release URL
if ! LATEST_URL=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep -i "browser_download_url.*${PLATFORM}_${ARCHITECTURE}.tar.gz" | cut -d '"' -f 4); then
    echo "Error: Could not find a release for $PLATFORM $ARCHITECTURE"
    exit 1
fi

if [ -z "$LATEST_URL" ]; then
    echo "Error: Could not find a release for $PLATFORM $ARCHITECTURE"
    exit 1
fi

echo "Downloading $BIN_NAME from $LATEST_URL ..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

# Download and extract (fail loudly on HTTP errors or truncated archives)
curl -fsSL "$LATEST_URL" -o release.tar.gz
tar -xzf release.tar.gz "$BIN_NAME"

# Determine install location and privilege level once
INSTALL_DIR="/usr/local/bin"
SUDO=""
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Requires sudo privileges to install to $INSTALL_DIR."
    SUDO="sudo"
fi

# Install with explicit mode (works with both mv and sudo paths)
if [ -n "$SUDO" ]; then
    $SUDO install -m 755 "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
    install -m 755 "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
fi

echo ""
echo "✅ Successfully installed $BIN_NAME to $INSTALL_DIR"
echo "Run 'git-user --help' to get started."
