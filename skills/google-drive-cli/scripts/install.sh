#!/usr/bin/env bash
set -euo pipefail

# This script downloads the latest release of drivectl from GitHub and extracts it to the current directory.

# Detect OS
OS_UNAME="$(uname -s)"
case "$OS_UNAME" in
  Linux) OS="Linux" ;;
  Darwin) OS="Darwin" ;;
  *) echo "Unsupported OS: $OS_UNAME"; exit 1 ;;
esac

# Detect Architecture
ARCH_UNAME="$(uname -m)"
case "$ARCH_UNAME" in
  x86_64|amd64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported Architecture: $ARCH_UNAME"; exit 1 ;;
esac

REPO="ghchinoy/drivectl"
FILENAME="drivectl_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${FILENAME}"

echo "Downloading drivectl from: ${URL}"
curl -sL "$URL" | tar -xz drivectl

chmod +x drivectl
echo "Success! drivectl is now installed in the current directory."
