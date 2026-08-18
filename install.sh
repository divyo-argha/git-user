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
CHECKSUMS_URL="$(printf '%s' "$RELEASE_JSON" | grep -i "browser_download_url.*checksums.txt" | cut -d '"' -f 4)"

if [ -z "$LATEST_URL" ]; then
    echo "Error: Could not find a release for $PLATFORM $ARCHITECTURE"
    exit 1
fi

if [ -z "$CHECKSUMS_URL" ]; then
    echo "Error: Release $LATEST_TAG has no checksums.txt — refusing to install an unverified binary."
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

# Pre-install check: capture existing version and check if an older install elsewhere in PATH shadows
PREV_VERSION="$("$BIN_NAME" --version 2>/dev/null | awk '{print $2}' || true)"
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

echo "Downloading $BIN_NAME $LATEST_TAG ($PLATFORM $ARCHITECTURE)..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

# Download and extract (fail loudly on HTTP errors or truncated archives)
curl -fsSL "$LATEST_URL" -o release.tar.gz

# Verify the download against the release's checksums.txt before extracting
# or installing anything. This is the actual security boundary for this
# installer: without it, a tampered or substituted release asset would be
# installed (and, when sudo is required below, run as root) with nothing to
# catch it.
curl -fsSL "$CHECKSUMS_URL" -o checksums.txt
ARCHIVE_NAME="$(basename "$LATEST_URL")"
EXPECTED_SHA256="$(grep " ${ARCHIVE_NAME}\$" checksums.txt | awk '{print $1}')"
if [ -z "$EXPECTED_SHA256" ]; then
    echo "Error: No checksum entry for $ARCHIVE_NAME in checksums.txt"
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_SHA256="$(sha256sum release.tar.gz | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_SHA256="$(shasum -a 256 release.tar.gz | awk '{print $1}')"
else
    echo "Error: neither sha256sum nor shasum is available to verify the download"
    exit 1
fi

if [ "$EXPECTED_SHA256" != "$ACTUAL_SHA256" ]; then
    echo "Error: checksum verification FAILED for $ARCHIVE_NAME"
    echo "  expected: $EXPECTED_SHA256"
    echo "  actual:   $ACTUAL_SHA256"
    exit 1
fi
echo "✅ Checksum verified"

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
printf "  \033[38;2;249;115;22m▄▀▀ █ ▀█▀\033[0m       \033[38;2;226;232;240m█ █ ▀▀▀ █▀▀ █▀▄\033[0m\n"
printf "  \033[38;2;249;115;22m█ ▄ █  █ \033[0m  \033[38;2;148;163;184m▄▄▄\033[0m  \033[38;2;226;232;240m█ █ ▀▀▄ █▀  █▀▀\033[0m\n"
printf "  \033[38;2;249;115;22m▀▀▀ ▀  ▀ \033[0m       \033[38;2;226;232;240m▀▀▀ ▀▀▀ ▀▀▀ ▀ ▀\033[0m\n"
printf "  \033[38;2;120;124;153mVersion %s\033[0m\n" "$LATEST_TAG"
echo ""

if [ -n "$PREV_VERSION" ] && [ "$PREV_VERSION" != "$LATEST_TAG" ]; then
    printf "╭──────────────────────────────────────────────╮\n"
    printf "│ \033[1;32m✨ Successfully updated git-user!\033[0m            │\n"
    printf "│                                              │\n"
    printf "│    \033[90m%-7s\033[0m \033[1;34m──▶\033[0m \033[1;32m%-7s\033[0m \033[32m(verified)\033[0m        │\n" "$PREV_VERSION" "$LATEST_TAG"
    printf "╰──────────────────────────────────────────────╯\n"
else
    printf "╭──────────────────────────────────────────────╮\n"
    printf "│ \033[1;32m✨ Successfully installed git-user!\033[0m          │\n"
    printf "│                                              │\n"
    printf "│    \033[1;32m%-7s\033[0m \033[32m(verified)\033[0m                 │\n" "$LATEST_TAG"
    printf "╰──────────────────────────────────────────────╯\n"
fi
printf "  \033[90mRun 'git-user' to launch the interactive dashboard.\033[0m\n\n"

# Verify the installed binary runs and reports the expected version
INSTALLED_VER="$("$INSTALL_DIR/$BIN_NAME" --version 2>/dev/null || true)"
if [ -z "$INSTALLED_VER" ]; then
    echo "⚠ Could not run the installed binary — check it with: $INSTALL_DIR/$BIN_NAME --version"
fi

# Post-install check: confirm which binary '$BIN_NAME' actually resolves to.
# An older install that appears earlier in PATH would shadow the new version.
RESOLVED="$(command -v "$BIN_NAME" 2>/dev/null || true)"
if [ -z "$RESOLVED" ]; then
    echo "Note: $INSTALL_DIR is not in this shell's PATH, so run:"
    echo "    $INSTALL_DIR/$BIN_NAME --help"
    echo "or add $INSTALL_DIR to your PATH and open a new terminal."
elif [ "$RESOLVED" != "$INSTALL_DIR/$BIN_NAME" ]; then
    echo ""
    echo "⚠  '$BIN_NAME' still resolves to an older copy at:"
    echo "     $RESOLVED"
    echo "   That copy shadows the new one on PATH. Remove it, then run:"
    echo "     rm \"$RESOLVED\""
    echo "     $BIN_NAME --version"
fi
