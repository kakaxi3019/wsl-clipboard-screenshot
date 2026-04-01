#!/bin/bash
set -euo pipefail

# Install script for wsl-clipboard-screenshot
# Usage: curl -fsSL https://raw.githubusercontent.com/kakaxi3019/wsl-clipboard-screenshot/master/scripts/install.sh | bash

REPO="kakaxi3019/wsl-clipboard-screenshot"
BINARY="wsl-clipboard-screenshot"
INSTALL_DIR="${HOME}/.local/bin"

info() {
    printf "\033[1;34m==>\033[0m %s\n" "$*"
}

error() {
    printf "\033[1;31merror:\033[0m %s\n" "$*" >&2
    exit 1
}

warn() {
    printf "\033[1;33mwarning:\033[0m %s\n" "$*" >&2
}

success() {
    printf "\033[1;32mdone:\033[0m %s\n" "$*"
}

detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64 | amd64)  echo "amd64" ;;
        aarch64 | arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $arch. Only amd64 and arm64 are supported." ;;
    esac
}

detect_os() {
    if grep -qiE "wsl|microsoft" /proc/version 2>/dev/null; then
        echo "linux"
        return
    fi
    error "WSL not detected. This tool only runs inside WSL."
}

get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local response

    if command -v curl &>/dev/null; then
        response=$(curl -fsSL "$url")
    elif command -v wget &>/dev/null; then
        response=$(wget -qO- "$url")
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    local version
    version=$(echo "$response" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        error "Failed to fetch latest version."
    fi

    echo "$version"
}

download() {
    local url="$1"
    local dest="$2"

    if command -v curl &>/dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -qO "$dest" "$url"
    else
        error "Neither curl nor wget found."
    fi
}

fix_path() {
    if echo ":$PATH:" | grep -q ":${INSTALL_DIR}:"; then
        return
    fi

    # Try /etc/profile.d/ first (works on most WSL distros without sudo)
    if [ -d "/etc/profile.d" ] && [ -w "/etc/profile.d" ]; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' > /etc/profile.d/wsl-clipboard-screenshot.sh
        success "Added PATH entry to /etc/profile.d/"
        echo "  PATH will be correct in new terminals."
        return
    fi

    # Update ~/.bashrc for future sessions
    if [ -f "${HOME}/.bashrc" ] && ! grep -q '\.local/bin' "${HOME}/.bashrc"; then
        cat >> "${HOME}/.bashrc" << 'PATHEOF'

# Added by wsl-clipboard-screenshot installer
if [ -d "$HOME/.local/bin" ]; then
    export PATH="$HOME/.local/bin:$PATH"
fi
PATHEOF
    fi

    # Export for current session
    export PATH="${INSTALL_DIR}:$PATH"

    success "Updated PATH for installation"
}

add_shell_autostart() {
    if [ -f "${HOME}/.bashrc" ] && grep -q "wsl-clipboard-screenshot start" "${HOME}/.bashrc"; then
        success "Shell auto-start already configured in ~/.bashrc"
        return
    fi

    cat >> "${HOME}/.bashrc" << 'STARTEOF'

# Auto-start wsl-clipboard-screenshot (added by installer)
wsl-clipboard-screenshot start --daemon 2>/dev/null
STARTEOF
    success "Added auto-start to ~/.bashrc"
}

setup_menu() {
    echo ""
    info "How would you like to auto-start wsl-clipboard-screenshot?"
    echo ""
    echo "  1) Add to shell profile (~/.bashrc)"
    echo "  2) Skip (I'll configure manually)"
    echo ""

    local choice
    printf "  Select [1-2]: "
    read -r choice

    case "$choice" in
        1)
            add_shell_autostart
            ;;
        2)
            info "Skipped. Run 'wsl-clipboard-screenshot start --daemon' manually."
            ;;
        *)
            warn "Invalid selection '${choice}' — skipping auto-start setup."
            ;;
    esac
}

main() {
    local os arch version version_stripped archive

    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    info "Fetching latest version..."
    version=$(get_latest_version)
    version_stripped="${version#v}"
    info "Latest version: ${version}"

    # Check if already installed (try both PATH and direct path)
    local installed_path="${INSTALL_DIR}/${BINARY}"
    if command -v "${BINARY}" &>/dev/null; then
        installed_path=$(command -v "${BINARY}")
    fi

    if [ -x "${installed_path}" ]; then
        local installed_version
        installed_version=$("${installed_path}" --version 2>&1 | awk '{print $NF}')
        if [ "${installed_version}" = "${version_stripped}" ]; then
            info "Already up to date (${version})."
            exit 0
        fi
        info "Updating from v${installed_version} to ${version}..."
    fi

    archive="${BINARY}_${version_stripped}_${os}_${arch}.tar.gz"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    local base_url="https://github.com/${REPO}/releases/download/${version}"

    info "Downloading ${archive}..."
    download "${base_url}/${archive}" "${tmpdir}/${archive}"

    info "Downloading checksums..."
    download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"

    info "Verifying checksum..."
    local expected actual
    expected=$(grep "${archive}" "${tmpdir}/checksums.txt" | awk '{print $1}')

    if [ -z "$expected" ]; then
        error "Could not find checksum for ${archive}"
    fi

    actual=$(sha256sum "${tmpdir}/${archive}" | awk '{print $1}')

    if [ "$expected" != "$actual" ]; then
        error "Checksum mismatch! Expected: ${expected}, Got: ${actual}"
    fi
    info "Checksum verified."

    info "Extracting ${BINARY}..."
    tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}" "${BINARY}"

    mkdir -p "${INSTALL_DIR}"
    mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"

    info "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

    fix_path

    echo ""
    info "Installation complete! Run '${BINARY} --help' to get started."
    setup_menu
}

main "$@"
