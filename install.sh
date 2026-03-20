#!/bin/sh
# install.sh — Install the finops CLI from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/helmcode/finops-cli/main/install.sh | sh
#
# The script detects your OS and architecture, downloads the latest release,
# and installs the binary to /usr/local/bin (or ~/.local/bin as fallback).

set -e

REPO="helmcode/finops-cli"
BINARY_NAME="finops"

# --- Helpers ---

info() {
    printf '[info] %s\n' "$1"
}

error() {
    printf '[error] %s\n' "$1" >&2
    exit 1
}

# --- Detect OS ---

detect_os() {
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*|Windows*)
            printf '[error] Windows is not supported by this installer.\n' >&2
            printf '        Please download the binary manually from:\n' >&2
            printf '        https://github.com/%s/releases/latest\n' "$REPO" >&2
            exit 1
            ;;
        *)
            error "Unsupported operating system: $os"
            ;;
    esac
}

# --- Detect Architecture ---

detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac
}

# --- Fetch latest version ---

get_latest_version() {
    url="https://api.github.com/repos/${REPO}/releases/latest"

    if command -v curl >/dev/null 2>&1; then
        response="$(curl -fsSL "$url")" || error "Failed to fetch latest release from GitHub API."
    elif command -v wget >/dev/null 2>&1; then
        response="$(wget -qO- "$url")" || error "Failed to fetch latest release from GitHub API."
    else
        error "Neither curl nor wget found. Please install one and try again."
    fi

    # Extract tag_name from JSON without requiring jq.
    version="$(printf '%s' "$response" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"

    if [ -z "$version" ]; then
        error "Could not determine latest version. GitHub API response may have changed or rate limit reached."
    fi

    # Strip leading 'v' if present (goreleaser uses the version without it in archive names).
    echo "$version" | sed 's/^v//'
}

# --- Determine install directory ---

get_install_dir() {
    if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ] && [ -w "$HOME/.local/bin" ]; then
        echo "$HOME/.local/bin"
    else
        mkdir -p "$HOME/.local/bin" 2>/dev/null || error "Cannot create $HOME/.local/bin. Please create it manually."
        echo "$HOME/.local/bin"
    fi
}

# --- Main ---

main() {
    info "Detecting platform..."
    os="$(detect_os)"
    arch="$(detect_arch)"
    info "Platform: ${os}/${arch}"

    info "Fetching latest release version..."
    version="$(get_latest_version)"
    info "Latest version: ${version}"

    archive="finops-cli_${version}_${os}_${arch}.tar.gz"
    download_url="https://github.com/${REPO}/releases/download/v${version}/${archive}"

    tmpdir="$(mktemp -d)" || error "Failed to create temporary directory."
    trap 'rm -rf "$tmpdir"' EXIT

    checksums_file="finops-cli_${version}_checksums.txt"
    checksums_url="https://github.com/${REPO}/releases/download/v${version}/${checksums_file}"

    info "Downloading ${archive}..."
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "${tmpdir}/${archive}" "$download_url" || error "Download failed. Check that version v${version} exists at: ${download_url}"
        curl -fsSL -o "${tmpdir}/${checksums_file}" "$checksums_url" || error "Failed to download checksums file."
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "${tmpdir}/${archive}" "$download_url" || error "Download failed. Check that version v${version} exists at: ${download_url}"
        wget -qO "${tmpdir}/${checksums_file}" "$checksums_url" || error "Failed to download checksums file."
    fi

    info "Verifying checksum..."
    expected_checksum="$(grep "${archive}" "${tmpdir}/${checksums_file}" | cut -d ' ' -f 1)"
    if [ -z "$expected_checksum" ]; then
        error "Checksum for ${archive} not found in checksums file."
    fi
    if command -v sha256sum >/dev/null 2>&1; then
        actual_checksum="$(sha256sum "${tmpdir}/${archive}" | cut -d ' ' -f 1)"
    elif command -v shasum >/dev/null 2>&1; then
        actual_checksum="$(shasum -a 256 "${tmpdir}/${archive}" | cut -d ' ' -f 1)"
    else
        error "No SHA256 tool found. Cannot verify download integrity."
    fi
    if [ "$expected_checksum" != "$actual_checksum" ]; then
        error "Checksum verification failed. Expected: ${expected_checksum}, Got: ${actual_checksum}"
    fi
    info "Checksum verified."

    info "Extracting binary..."
    tar --no-same-owner -xzf "${tmpdir}/${archive}" -C "$tmpdir" || error "Failed to extract archive."

    if [ ! -f "${tmpdir}/${BINARY_NAME}" ]; then
        error "Binary '${BINARY_NAME}' not found in archive. The release archive may have an unexpected structure."
    fi

    install_dir="$(get_install_dir)"
    info "Installing to ${install_dir}/${BINARY_NAME}..."

    mv "${tmpdir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}" || error "Failed to install binary to ${install_dir}. Try running with sudo."
    chmod 755 "${install_dir}/${BINARY_NAME}"

    info "Successfully installed finops v${version} to ${install_dir}/${BINARY_NAME}"

    # Check if the install directory is in PATH.
    case ":${PATH}:" in
        *":${install_dir}:"*) ;;
        *)
            printf '\n[warn] %s is not in your PATH.\n' "$install_dir"
            printf '       Add it with:  export PATH="%s:$PATH"\n' "$install_dir"
            ;;
    esac
}

main
