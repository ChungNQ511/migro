#!/bin/bash
# Migro Database Migration Tool Installer
# Usage: curl -sSL https://raw.githubusercontent.com/ChungNQ511/migro/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="migro"
GITHUB_REPO="ChungNQ511/migro"
INSTALL_DIR="$HOME/.local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case $OS in
    linux)
        PLATFORM="linux"
        ;;
    darwin)
        PLATFORM="darwin"
        ;;
    *)
        echo -e "${RED}❌ Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

BINARY_NAME_WITH_PLATFORM="${BINARY_NAME}-${PLATFORM}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_NAME_WITH_PLATFORM="${BINARY_NAME_WITH_PLATFORM}.exe"
fi

echo -e "${BLUE}🚀 Installing Migro Database Migration Tool${NC}"
echo -e "${YELLOW}📊 Detected platform: ${PLATFORM}-${ARCH}${NC}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Get latest release
echo -e "${YELLOW}🔍 Fetching latest release...${NC}"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}❌ Failed to fetch latest version${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Latest version: $LATEST_VERSION${NC}"

# Download URL
DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$LATEST_VERSION/${BINARY_NAME_WITH_PLATFORM}"

# Download binary
echo -e "${YELLOW}⬇️  Downloading from: $DOWNLOAD_URL${NC}"
if curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"; then
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    echo -e "${GREEN}✅ Successfully installed $BINARY_NAME to $INSTALL_DIR${NC}"
else
    echo -e "${RED}❌ Failed to download binary${NC}"
    exit 1
fi

# Check if install directory is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo -e "${YELLOW}⚠️  $INSTALL_DIR is not in your PATH${NC}"
    echo -e "${YELLOW}   Add the following line to your shell profile (.bashrc, .zshrc, etc.):${NC}"
    echo -e "${BLUE}   export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
    echo
fi

# Test installation
if "$INSTALL_DIR/$BINARY_NAME" --help >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Installation successful!${NC}"
    echo
    echo -e "${BLUE}🎉 You can now use migro:${NC}"
    echo -e "   $BINARY_NAME --help"
    echo -e "   $BINARY_NAME status"
    echo
    echo -e "${YELLOW}📄 Next steps:${NC}"
    echo -e "   1. Create migro.yaml configuration file"
    echo -e "   2. Run: $BINARY_NAME status"
    echo
else
    echo -e "${RED}❌ Installation verification failed${NC}"
    exit 1
fi 