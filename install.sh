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

# Get the latest release
if ! RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest"); then
    echo "Error: Could not fetch release information"
    exit 1
fi

LATEST_URL="$(printf '%s' "$RELEASE_JSON" | grep -i "browser_download_url.*${PLATFORM}_${ARCHITECTURE}.tar.gz" | cut -d '"' -f 4)"
LATEST_TAG="$(printf '%s' "$RELEASE_JSON" | grep -i '"tag_name"' | cut -d '"' -f 4)"

if [ -z "$LATEST_URL" ]; then
    echo "Error: Could not find a release for $PLATFORM $ARCHITECTURE"
    exit 1
fi

echo "Latest release: $LATEST_TAG"

# Determine install location and privilege level once
INSTALL_DIR="/usr/local/bin"
SUDO=""
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Requires sudo privileges to install to $INSTALL_DIR."
    SUDO="sudo"
fi

# Pre-install check: an older install elsewhere in PATH would shadow the new binary
EXISTING="$(command -v "$BIN_NAME" 2>/dev/null || true)"
if [ -n "$EXISTING" ] && [ "$EXISTING" != "$INSTALL_DIR/$BIN_NAME" ]; then
    echo ""
    echo "⚠  A previous version of $BIN_NAME is installed at:"
    echo "     $EXISTING"
    echo "   That path comes before $INSTALL_DIR in your PATH, so '$BIN_NAME'"
    echo "   would keep running the OLD version after this install."
    echo ""
    echo "   To use the new version, remove the old install and re-run this script:"
    echo "     rm \"$EXISTING\""
    echo "   (if installed via npm instead: npm uninstall -g git-userhub)"
    echo ""
    echo "   Alternatively, make sure $INSTALL_DIR comes before that path in PATH."
    echo ""
fi

echo "Downloading $BIN_NAME from $LATEST_URL ..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

# Download and extract (fail loudly on HTTP errors or truncated archives)
curl -fsSL "$LATEST_URL" -o release.tar.gz
tar -xzf release.tar.gz "$BIN_NAME"

# Install with explicit mode. Unlink any previous version first: writing over
# a *running* binary fails with "text file busy" on macOS (BSD install),
# while unlink+install works on both macOS and Linux.
if [ -n "$SUDO" ]; then
    $SUDO rm -f "$INSTALL_DIR/$BIN_NAME"
    $SUDO install -m 755 "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
    rm -f "$INSTALL_DIR/$BIN_NAME"
    install -m 755 "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
fi

echo ""
echo "✅ Successfully installed $BIN_NAME $LATEST_TAG to $INSTALL_DIR"

# Post-install check: confirm which binary '$BIN_NAME' actually resolves to.
# An older install that appears earlier in PATH would shadow the new version.
RESOLVED="$(command -v "$BIN_NAME" 2>/dev/null || true)"
if [ "$RESOLVED" = "$INSTALL_DIR/$BIN_NAME" ]; then
    echo "Run '$BIN_NAME --help' to get started."
    echo "(In terminals opened before this install, run 'hash -r' first.)"
elif [ -z "$RESOLVED" ]; then
    echo "Note: $INSTALL_DIR is not in this shell's PATH, so run:"
    echo "    $INSTALL_DIR/$BIN_NAME --help"
    echo "or add $INSTALL_DIR to your PATH and open a new terminal."
else
    echo ""
    echo "⚠  '$BIN_NAME' still resolves to an older copy at:"
    echo "     $RESOLVED"
    echo "   That copy shadows the new one on PATH. Remove it, then run:"
    echo "     rm \"$RESOLVED\""
    echo "     $BIN_NAME --version"
fi
