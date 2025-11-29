#!/bin/sh
# pipeboard uninstaller
# Usage: curl -sSL https://raw.githubusercontent.com/blackwell-systems/pipeboard/main/uninstall.sh | sh

set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/pipeboard"

echo "Uninstalling pipeboard..."

# Remove binary
if [ -f "${INSTALL_DIR}/pipeboard" ]; then
    if [ -w "$INSTALL_DIR" ]; then
        rm "${INSTALL_DIR}/pipeboard"
    else
        echo "Removing ${INSTALL_DIR}/pipeboard (requires sudo)..."
        sudo rm "${INSTALL_DIR}/pipeboard"
    fi
    echo "Removed ${INSTALL_DIR}/pipeboard"
else
    echo "Binary not found at ${INSTALL_DIR}/pipeboard"
fi

# Ask about config removal
if [ -d "$CONFIG_DIR" ]; then
    echo ""
    echo "Config directory found at ${CONFIG_DIR}"
    echo "Contents:"
    ls -la "$CONFIG_DIR" 2>/dev/null || true
    echo ""
    printf "Remove config directory? [y/N] "
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            rm -rf "$CONFIG_DIR"
            echo "Removed ${CONFIG_DIR}"
            ;;
        *)
            echo "Kept ${CONFIG_DIR}"
            ;;
    esac
else
    echo "No config directory found at ${CONFIG_DIR}"
fi

echo ""
echo "pipeboard uninstalled."
