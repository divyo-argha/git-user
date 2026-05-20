package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/ui"
)

func RunUpdate() error {
	// Check if git-user is installed
	execPath, err := os.Executable()
	if err != nil {
		ui.Errorf("Could not determine installation path")
		return err
	}

	ui.Info(fmt.Sprintf("Updating git-user from %s...", execPath))

	// Determine install directory
	installDir := filepath.Dir(execPath)

	// Download and run install script
	tempScript := filepath.Join(os.TempDir(), "git-user-update.sh")
	defer os.Remove(tempScript)

	// Create update script
	script := `#!/bin/bash
set -e

INSTALL_DIR="` + installDir + `"
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Download pre-built binary from GitHub releases
RELEASE_URL="https://api.github.com/repos/divyo-argha/git-user/releases/latest"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
esac

DOWNLOAD_URL=$(curl -s "$RELEASE_URL" | grep "browser_download_url.*${OS}_${ARCH}" | cut -d '"' -f 4 | head -n 1)

if [ -n "$DOWNLOAD_URL" ]; then
    curl -sSfL "$DOWNLOAD_URL" -o git-user.tar.gz
    tar -xzf git-user.tar.gz
    BINARY="git-user"
else
    echo "Building from source..."
    git clone --depth 1 https://github.com/divyo-argha/git-user.git .
    go mod download
    go build -ldflags="-s -w" -o git-user .
    BINARY="git-user"
fi

# Backup current binary
if [ -f "$INSTALL_DIR/git-user" ]; then
    cp "$INSTALL_DIR/git-user" "$INSTALL_DIR/git-user.bak"
fi

# Install new binary
if [ -w "$INSTALL_DIR" ]; then
    cp "$BINARY" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/git-user"
else
    sudo cp "$BINARY" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/git-user"
fi

# Cleanup
cd /
rm -rf "$TEMP_DIR"

echo "✓ Update complete"
`

	if err := os.WriteFile(tempScript, []byte(script), 0755); err != nil {
		ui.Errorf("Failed to create update script: %v", err)
		return err
	}

	// Execute update script
	cmd := exec.Command("bash", tempScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		ui.Errorf("Update failed: %v", err)
		return err
	}

	fmt.Printf("\n%s✨ git-user updated successfully%s\n", "\033[32m", "\033[0m")
	fmt.Printf("%sRestart your terminal or run: source ~/.zshrc%s\n", "\033[36m", "\033[0m")

	return nil
}
