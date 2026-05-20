#!/bin/bash
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

# Check if already installed
if command -v git-user &> /dev/null; then
    CURRENT_VERSION=$(git-user --version 2>&1 | grep -oP 'v\d+\.\d+\.\d+' || echo "unknown")
    MODE="UPDATE"
    echo -e "${CYAN}╔════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║   Git-User Updater                     ║${NC}"
    echo -e "${CYAN}║   Current: $CURRENT_VERSION                      ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════╝${NC}\n"
else
    MODE="INSTALL"
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║     Git-User Installer                ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}\n"
fi

# Detect OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# Determine install directory
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
    NEEDS_SUDO=false
else
    INSTALL_DIR="/usr/local/bin"
    NEEDS_SUDO=true
fi

# Check prerequisites
echo -e "${BLUE}[1/5]${NC} Checking prerequisites..."

if ! command -v git &> /dev/null; then
    echo -e "${RED}✖ Git is not installed${NC}"
    echo "Please install git first: https://git-scm.com/downloads"
    exit 1
fi
echo -e "${GREEN}✓ Git found${NC}"

if ! command -v ssh-keygen &> /dev/null; then
    echo -e "${YELLOW}⚠ ssh-keygen not found (optional)${NC}"
else
    echo -e "${GREEN}✓ ssh-keygen found${NC}"
fi

# Download or build
echo -e "\n${BLUE}[2/5]${NC} Getting latest git-user..."

TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Try to download pre-built binary from GitHub releases
RELEASE_URL="https://api.github.com/repos/divyo-argha/git-user/releases/latest"
DOWNLOAD_URL=$(curl -s "$RELEASE_URL" | grep "browser_download_url.*${OS}_${ARCH}" | cut -d '"' -f 4 | head -n 1)

if [ -n "$DOWNLOAD_URL" ]; then
    echo "Downloading pre-built binary..."
    curl -sSfL "$DOWNLOAD_URL" -o git-user.tar.gz
    tar -xzf git-user.tar.gz
    BINARY="git-user"
else
    echo "Building from source..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${YELLOW}Go not found. Installing Go...${NC}"
        GO_VERSION="1.21.0"
        GO_FILE="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
        curl -sSfL "https://go.dev/dl/${GO_FILE}" -o go.tar.gz
        
        if [ "$NEEDS_SUDO" = true ]; then
            sudo tar -C /usr/local -xzf go.tar.gz
        else
            tar -C /usr/local -xzf go.tar.gz
        fi
        
        export PATH="/usr/local/go/bin:$PATH"
        echo -e "${GREEN}✓ Go installed${NC}"
    fi
    
    # Clone and build
    git clone --depth 1 https://github.com/divyo-argha/git-user.git .
    go mod download
    go build -ldflags="-s -w" -o git-user .
    BINARY="git-user"
fi

echo -e "${GREEN}✓ Binary ready${NC}"

# Install binary
echo -e "\n${BLUE}[3/5]${NC} Installing to $INSTALL_DIR..."

if [ "$NEEDS_SUDO" = true ]; then
    sudo cp "$BINARY" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/git-user"
else
    cp "$BINARY" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/git-user"
fi

echo -e "${GREEN}✓ Installed${NC}"

# Configure PATH automatically (only for fresh install)
if [ "$MODE" = "INSTALL" ]; then
    echo -e "\n${BLUE}[4/5]${NC} Configuring PATH..."

    SHELL_RC=""
    SHELL_NAME=$(basename "$SHELL")

    case "$SHELL_NAME" in
        zsh)
            SHELL_RC="$HOME/.zshrc"
            ;;
        bash)
            if [ -f "$HOME/.bash_profile" ]; then
                SHELL_RC="$HOME/.bash_profile"
            else
                SHELL_RC="$HOME/.bashrc"
            fi
            ;;
        fish)
            SHELL_RC="$HOME/.config/fish/config.fish"
            mkdir -p "$(dirname "$SHELL_RC")"
            ;;
        *)
            echo -e "${YELLOW}⚠ Unknown shell: $SHELL_NAME${NC}"
            ;;
    esac

    if [ -n "$SHELL_RC" ]; then
        if ! grep -q "git-user" "$SHELL_RC" 2>/dev/null; then
            echo "" >> "$SHELL_RC"
            echo "# Added by git-user installer" >> "$SHELL_RC"
            if [ "$SHELL_NAME" = "fish" ]; then
                echo "set -gx PATH $INSTALL_DIR \$PATH" >> "$SHELL_RC"
            else
                echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
            fi
            echo -e "${GREEN}✓ PATH configured in $SHELL_RC${NC}"
        else
            echo -e "${GREEN}✓ PATH already configured${NC}"
        fi
        
        # Source the file to apply changes immediately
        if [ "$SHELL_NAME" != "fish" ]; then
            export PATH="$INSTALL_DIR:$PATH"
        fi
    fi
    
    STEP="[5/5]"
else
    STEP="[4/5]"
fi

# Cleanup
cd /
rm -rf "$TEMP_DIR"

# Verify installation
echo -e "\n${BLUE}$STEP${NC} Verifying installation..."

if command -v git-user &> /dev/null; then
    NEW_VERSION=$(git-user --version 2>&1 | grep -oP 'v\d+\.\d+\.\d+' || echo "installed")
    echo -e "${GREEN}✓ git-user $NEW_VERSION${NC}"
else
    echo -e "${YELLOW}⚠ git-user not found in PATH${NC}"
    echo "You may need to restart your terminal or run:"
    echo "  source $SHELL_RC"
fi

# Success message
if [ "$MODE" = "UPDATE" ]; then
    echo -e "\n${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   Update Complete! ✨                  ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════╝${NC}\n"
else
    echo -e "\n${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   Installation Complete! 🎉            ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════╝${NC}\n"

    echo -e "${BLUE}Quick Start:${NC}"
    echo "  1. Restart your terminal (or run: source $SHELL_RC)"
    echo "  2. Create your first identity:"
    echo -e "     ${YELLOW}git-user register${NC}"
    echo ""
    echo "  3. Switch between identities:"
    echo -e "     ${YELLOW}git-user switch <name>${NC}"
    echo ""
    echo "  4. Check your setup:"
    echo -e "     ${YELLOW}git-user doctor${NC}"
    echo ""
    echo "  5. Get help:"
    echo -e "     ${YELLOW}git-user --help${NC}"
    echo ""
fi
