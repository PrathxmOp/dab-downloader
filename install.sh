#!/bin/bash

# DAB Downloader Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/PrathxmOp/dab-downloader/dev/install.sh | bash

set -e

REPO="PrathxmOp/dab-downloader"
BINARY_NAME="dab-downloader"

# Detect Termux
IS_TERMUX=false
if [ -n "$TERMUX_VERSION" ]; then
    IS_TERMUX=true
    INSTALL_DIR="$PREFIX/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

printf "${BLUE}🎵 DAB Downloader Installer${NC}\n"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    linux*)     OS='linux' ;;
    darwin*)    OS='darwin' ;;
    msys*|cygwin*|mingw*) OS='windows' ;;
    *)          printf "${RED}❌ Unsupported OS: ${OS}${NC}\n"; exit 1 ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) ARCH='amd64' ;;
    armv8*|aarch64) ARCH='arm64' ;;
    i386|i686) ARCH='386' ;;
    *)          printf "${RED}❌ Unsupported architecture: ${ARCH}${NC}\n"; exit 1 ;;
esac

printf "🔍 Detected: ${OS}-${ARCH}\n"

# Fetch latest release info from GitHub API
printf "📡 Fetching latest release info...\n"
RELEASE_JSON=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest")
LATEST_VERSION=$(echo "${RELEASE_JSON}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    printf "${RED}❌ Could not determine latest version. Check your internet connection or GitHub API limits.${NC}\n"
    exit 1
fi

printf "✨ Latest version: ${LATEST_VERSION}\n"

# Construct download URL
# The assets are named like 'dab-downloader-linux-amd64' (raw binary)
FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    FILENAME="${FILENAME}.exe"
fi

DOWNLOAD_URL=$(echo "${RELEASE_JSON}" | grep "browser_download_url" | grep "${FILENAME}" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    printf "${RED}❌ Could not find a download URL for ${FILENAME}.${NC}\n"
    printf "Please check the releases page: https://github.com/${REPO}/releases\n"
    exit 1
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
cd "${TMP_DIR}"

# Download
printf "📥 Downloading ${FILENAME}...\n"
curl -L -o "${BINARY_NAME}" "${DOWNLOAD_URL}"

# Install
printf "🚀 Installing to ${INSTALL_DIR}...\n"
mv "${BINARY_NAME}" "${INSTALL_DIR}/"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Cleanup
rm -rf "${TMP_DIR}"

printf "${GREEN}✅ Successfully installed DAB Downloader ${LATEST_VERSION}!${NC}\n"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    printf "${RED}⚠️  Warning: ${INSTALL_DIR} is not in your PATH.${NC}\n"
    printf "You may need to add it to your shell configuration (e.g., .bashrc or .zshrc):\n"
    printf "${BLUE}export PATH=\$PATH:${INSTALL_DIR}${NC}\n"
else
    printf "Run '${BINARY_NAME} version' to verify.\n"
fi
