#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
DATA_DIR="/var/lib/notifyd"
CONFIG_DIR="/etc/notifyd"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -sL https://api.github.com/repos/notifyd-eng/notifyd/releases/latest | grep tag_name | cut -d '"' -f 4)
fi

TARBALL="notifyd_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/notifyd-eng/notifyd/releases/download/${VERSION}/${TARBALL}"

echo "Installing notifyd ${VERSION} (${OS}/${ARCH})..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sL "$URL" -o "$TMP/$TARBALL"
tar xzf "$TMP/$TARBALL" -C "$TMP"

sudo install -m 755 "$TMP/notifyd" "$INSTALL_DIR/notifyd"

if [ ! -d "$DATA_DIR" ]; then
    sudo mkdir -p "$DATA_DIR"
    echo "Created data directory: $DATA_DIR"
fi

if [ ! -d "$CONFIG_DIR" ]; then
    sudo mkdir -p "$CONFIG_DIR"
    if [ -f "$TMP/config.example.yaml" ]; then
        sudo cp "$TMP/config.example.yaml" "$CONFIG_DIR/config.yaml"
    fi
    echo "Created config directory: $CONFIG_DIR"
fi

echo "notifyd ${VERSION} installed to ${INSTALL_DIR}/notifyd"
