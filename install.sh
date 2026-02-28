#!/bin/sh
set -e

# POSIX compliant way to get HOME if not set
if [ -z "$HOME" ]; then
    HOME=$(eval echo ~)
fi

GIOS_BIN_DIR="$HOME/.gios/bin"

echo "=> Compiling gios CLI..."
go build -o gios main.go

echo "=> Creating installation directory at $GIOS_BIN_DIR..."
mkdir -p "$GIOS_BIN_DIR"

echo "=> Installing gios to $GIOS_BIN_DIR..."
mv gios "$GIOS_BIN_DIR/gios"

echo "=> Mission accomplished!"

# Check if PATH contains GIOS_BIN_DIR using case statement (POSIX compliant way)
case ":$PATH:" in
    *":$GIOS_BIN_DIR:"*)
        echo "The 'gios' tool has been successfully installed or updated!"
        ;;
    *)
        echo "NOTE: Make sure to have $GIOS_BIN_DIR in your PATH environment variable."
        echo "You can add it by appending the following line to your ~/.profile, ~/.zshrc or ~/.bashrc:"
        echo "export PATH=\"\$PATH:$GIOS_BIN_DIR\""
        ;;
esac
