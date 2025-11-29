#!/bin/sh
# pipeboard installer
# Usage: curl -sSL https://raw.githubusercontent.com/blackwell-systems/pipeboard/main/install.sh | sh

set -e

REPO="blackwell-systems/pipeboard"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Get latest release tag
LATEST=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "Failed to fetch latest release"
    exit 1
fi

echo "Installing pipeboard ${LATEST} (${OS}/${ARCH})..."

# Download URL
FILENAME="pipeboard_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download and extract
echo "Downloading ${URL}..."
HTTP_CODE=$(curl -sSL -w "%{http_code}" "$URL" -o "${TMP_DIR}/${FILENAME}")
if [ "$HTTP_CODE" != "200" ]; then
    echo "Error: Failed to download release (HTTP $HTTP_CODE)"
    echo "URL: $URL"
    echo ""
    echo "The release may not exist yet. Try installing via Go instead:"
    echo "  go install github.com/blackwell-systems/pipeboard@latest"
    exit 1
fi

# Verify it's a valid gzip file
if ! file "${TMP_DIR}/${FILENAME}" | grep -q "gzip"; then
    echo "Error: Downloaded file is not a valid gzip archive"
    echo "The release asset may not exist. Try installing via Go instead:"
    echo "  go install github.com/blackwell-systems/pipeboard@latest"
    exit 1
fi

tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/pipeboard" "${INSTALL_DIR}/pipeboard"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${TMP_DIR}/pipeboard" "${INSTALL_DIR}/pipeboard"
fi

chmod +x "${INSTALL_DIR}/pipeboard"

echo "Installed pipeboard to ${INSTALL_DIR}/pipeboard"
echo ""
echo "Run 'pipeboard doctor' to verify your setup."
