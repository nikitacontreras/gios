#!/bin/bash
set -e

# GIOS Easy Installer
# Installs gios binary to ~/.gios/bin

REPO="nikitacontreras/gios"
GIOS_DIR="$HOME/.gios"
BIN_DIR="$GIOS_DIR/bin"

echo "--------------------------------------------------"
echo "   🚀 GIOS - Universal Installer"
echo "--------------------------------------------------"

# Detect OS
OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS_NAME" in
  linux*)   OS="linux" ;;
  darwin*)  OS="darwin" ;;
  msys*|mingw*) OS="windows" ;;
  *)        echo "Unsupported OS: $OS_NAME"; exit 1 ;;
esac

# Detect Arch
ARCH_NAME="$(uname -m)"
case "$ARCH_NAME" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)            echo "Unsupported architecture: $ARCH_NAME"; exit 1 ;;
esac

# Get latest release from GitHub API
echo "[1/3] Checking latest release info..."
LATEST_RELEASE=$(curl -s https://api.github.com/repos/$REPO/releases/latest)

# Portable way to extract tag_name
VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | head -n 1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo " [!] Stable release not found, falling back to nightly..."
  VERSION="latest"
fi

BINARY_NAME="gios-$OS-$ARCH"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="$BINARY_NAME.exe"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_NAME"

echo "[2/3] Downloading GIOS $VERSION ($OS/$ARCH)..."
mkdir -p "$BIN_DIR"

if command -v curl >/dev/null 2>&1; then
  curl -L "$DOWNLOAD_URL" -o "$BIN_DIR/gios"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$BIN_DIR/gios" "$DOWNLOAD_URL"
else
  echo "Error: curl or wget is required."
  exit 1
fi

chmod +x "$BIN_DIR/gios"

echo "[3/3] Finalizing installation..."

# PATH Setup
PATH_ADDED=false
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  SHELL_NAME=$(basename "$SHELL")
  SHELL_RC=""
  
  case "$SHELL_NAME" in
    zsh)  SHELL_RC="$HOME/.zshrc" ;;
    bash) [ -f "$HOME/.bash_profile" ] && SHELL_RC="$HOME/.bash_profile" || SHELL_RC="$HOME/.bashrc" ;;
    *)    SHELL_RC="$HOME/.profile" ;;
  esac
  
  if [ -f "$SHELL_RC" ]; then
    if ! grep -q "$BIN_DIR" "$SHELL_RC"; then
        echo "" >> "$SHELL_RC"
        echo "# GIOS PATH" >> "$SHELL_RC"
        echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$SHELL_RC"
        PATH_ADDED=true
    fi
  fi
fi

echo "--------------------------------------------------"
echo "✅ GIOS installed successfully to $BIN_DIR/gios"
if [ "$PATH_ADDED" = true ]; then
  echo "   (PATH updated in $SHELL_RC)"
  echo "   Please run: source $SHELL_RC"
fi
echo "   Try running: gios doctor"
echo "--------------------------------------------------"
