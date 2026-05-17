#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Git-User Installer ===${NC}\n"

# Detect OS and Architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Map architecture
if [[ "$ARCH" == "arm64" ]]; then
    GO_ARCH="arm64"
elif [[ "$ARCH" == "x86_64" ]]; then
    GO_ARCH="amd64"
else
    GO_ARCH="$ARCH"
fi

# Determine install directory
INSTALL_DIR="/usr/local/bin"

# Function to check if command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Function to install Go
install_go() {
    echo -e "${BLUE}Installing Go...${NC}"
    
    if [[ "$OS" == "Darwin" ]]; then
        # macOS
        if command_exists brew; then
            echo "Using Homebrew to install Go..."
            brew install go
        else
            # Manual installation for macOS
            GO_VERSION="1.21.0"
            GO_FILE="go${GO_VERSION}.darwin-${GO_ARCH}.tar.gz"
            echo "Downloading Go ${GO_VERSION}..."
            curl -sSfL "https://go.dev/dl/${GO_FILE}" -o "/tmp/${GO_FILE}"
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "/tmp/${GO_FILE}"
            rm "/tmp/${GO_FILE}"
        fi
    elif [[ "$OS" == "Linux" ]]; then
        # Linux
        GO_VERSION="1.21.0"
        GO_FILE="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
        echo "Downloading Go ${GO_VERSION}..."
        curl -sSfL "https://go.dev/dl/${GO_FILE}" -o "/tmp/${GO_FILE}"
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "/tmp/${GO_FILE}"
        rm "/tmp/${GO_FILE}"
    else
        echo -e "${RED}Unsupported OS: $OS${NC}"
        echo "Please install Go manually from https://golang.org/dl/"
        exit 1
    fi
    
    # Add Go to PATH
    export PATH="/usr/local/go/bin:$PATH"
    echo -e "${GREEN}✓ Go installed successfully${NC}"
}

# Check if Go is installed
if ! command_exists go; then
    echo -e "${YELLOW}Go is not installed${NC}"
    install_go
else
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✓ Go is already installed: $GO_VERSION${NC}"
fi

# Check if git is installed
if ! command_exists git; then
    echo -e "${RED}Git is not installed. Please install git first.${NC}"
    exit 1
fi

# Check if ssh-keygen is available
if ! command_exists ssh-keygen; then
    echo -e "${YELLOW}Warning: ssh-keygen not found. Some features may not work.${NC}"
fi

echo -e "\n${BLUE}Step 1: Cloning repository...${NC}"
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Try to clone from GitHub, fallback to local if available
if git clone https://github.com/divyo-argha/git-user.git . 2>/dev/null; then
    echo -e "${GREEN}✓ Repository cloned${NC}"
else
    echo -e "${YELLOW}Could not clone from GitHub. Checking for local repository...${NC}"
    if [[ -d "/Users/bobdylan/Divyo/git-user" ]]; then
        cp -r /Users/bobdylan/Divyo/git-user/* .
        echo -e "${GREEN}✓ Using local repository${NC}"
    else
        echo -e "${RED}Could not find repository${NC}"
        exit 1
    fi
fi

echo -e "\n${BLUE}Step 2: Building git-user...${NC}"
go mod tidy > /dev/null 2>&1
go build -o git-user
echo -e "${GREEN}✓ Build complete${NC}"

echo -e "\n${BLUE}Step 3: Installing to $INSTALL_DIR...${NC}"
sudo cp git-user "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/git-user"
echo -e "${GREEN}✓ Installed to $INSTALL_DIR/git-user${NC}"

echo -e "\n${BLUE}Step 4: Updating PATH...${NC}"
SHELL_RC=""
if [[ "$SHELL" == *"zsh"* ]]; then
    SHELL_RC="$HOME/.zshrc"
elif [[ "$SHELL" == *"bash"* ]]; then
    SHELL_RC="$HOME/.bashrc"
elif [[ "$SHELL" == *"fish"* ]]; then
    SHELL_RC="$HOME/.config/fish/config.fish"
fi

if [[ -n "$SHELL_RC" ]]; then
    if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
        if [[ "$SHELL" == *"fish"* ]]; then
            echo "set -gx PATH $INSTALL_DIR \$PATH" >> "$SHELL_RC"
        else
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
        fi
        echo -e "${GREEN}✓ Added $INSTALL_DIR to PATH in $SHELL_RC${NC}"
    else
        echo -e "${GREEN}✓ $INSTALL_DIR already in PATH${NC}"
    fi
fi

# Cleanup
cd /
rm -rf "$TEMP_DIR"

echo -e "\n${GREEN}✓✓✓ Installation complete! ✓✓✓${NC}\n"
echo -e "${BLUE}Quick start:${NC}"
echo "  1. Reload your shell:"
if [[ -n "$SHELL_RC" ]]; then
    echo "     source $SHELL_RC"
fi
echo ""
echo "  2. Create your first identity:"
echo "     git-user register"
echo ""
echo "  3. List all identities:"
echo "     git-user list"
echo ""
echo "  4. Switch between identities:"
echo "     git-user switch <name>"
echo ""
echo "  5. Check your setup:"
echo "     git-user doctor"
echo ""
echo "  6. Get help:"
echo "     git-user --help"
echo ""
