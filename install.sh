#!/bin/bash
set -e

# Installation directory for the current user
GIOS_BIN_DIR="$HOME/.gios/bin"

echo "=> Compiling gios CLI..."
go build -o gios main.go

echo "=> Creating installation directory at $GIOS_BIN_DIR..."
mkdir -p "$GIOS_BIN_DIR"

echo "=> Installing gios to $GIOS_BIN_DIR..."
mv gios "$GIOS_BIN_DIR/gios"

echo "=> Mission accomplished!"
if [[ ":$PATH:" != *":$GIOS_BIN_DIR:"* ]]; then
    echo "NOTE: Make sure to have $GIOS_BIN_DIR in your PATH environment variable."
    echo "You can add it by appending the following line to your ~/.zshrc or ~/.bashrc:"
    echo "export PATH=\$PATH:$GIOS_BIN_DIR"
else
    echo "The 'gios' tool has been successfully installed or updated!"
fi
