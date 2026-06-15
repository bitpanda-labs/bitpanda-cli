#!/bin/sh
set -e

REPO="bitpanda-labs/bitpanda-cli"
BINARY="bp"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux)  OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *)
        echo "Error: unsupported operating system: $OS" >&2
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

if [ "$OS" = "windows" ] && [ "$ARCH" = "arm64" ]; then
    echo "Error: windows/arm64 is not supported" >&2
    exit 1
fi

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version" >&2
    exit 1
fi
echo "Latest version: v${VERSION}"

# Build download URL
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    EXT="zip"
fi
FILENAME="bitpanda-cli_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

# Download and install
TMPDIR_INSTALL=$(mktemp -d)
trap 'rm -rf "$TMPDIR_INSTALL"' EXIT

echo "Downloading ${FILENAME}..."
curl -fsSL "$URL" -o "${TMPDIR_INSTALL}/${FILENAME}"

# Verify checksum before extracting/installing
echo "Verifying checksum..."
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"
if ! curl -fsSL "$CHECKSUMS_URL" -o "${TMPDIR_INSTALL}/checksums.txt"; then
    echo "Error: checksum verification failed for ${FILENAME}: could not download checksums.txt" >&2
    exit 1
fi

# Determine which SHA-256 tool is available
if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_HASH=$(sha256sum "${TMPDIR_INSTALL}/${FILENAME}" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_HASH=$(shasum -a 256 "${TMPDIR_INSTALL}/${FILENAME}" | cut -d' ' -f1)
else
    echo "Error: no SHA-256 tool found (need 'sha256sum' or 'shasum')" >&2
    exit 1
fi

# Extract the expected hash for this filename from checksums.txt
# Format per line: <hash>  <filename>
EXPECTED_HASH=$(grep " ${FILENAME}\$" "${TMPDIR_INSTALL}/checksums.txt" | cut -d' ' -f1)
if [ -z "$EXPECTED_HASH" ]; then
    echo "Error: checksum verification failed for ${FILENAME}: not found in checksums.txt" >&2
    exit 1
fi

if [ "$ACTUAL_HASH" != "$EXPECTED_HASH" ]; then
    echo "Error: checksum verification failed for ${FILENAME}" >&2
    echo "  expected: ${EXPECTED_HASH}" >&2
    echo "  actual:   ${ACTUAL_HASH}" >&2
    exit 1
fi
echo "Checksum verified."

echo "Extracting..."
if [ "$EXT" = "zip" ]; then
    unzip -q "${TMPDIR_INSTALL}/${FILENAME}" -d "$TMPDIR_INSTALL"
else
    tar -xzf "${TMPDIR_INSTALL}/${FILENAME}" -C "$TMPDIR_INSTALL"
fi

echo "Installing ${BINARY} to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
    install -m 755 "${TMPDIR_INSTALL}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    sudo install -m 755 "${TMPDIR_INSTALL}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Successfully installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"
${INSTALL_DIR}/${BINARY} --version
