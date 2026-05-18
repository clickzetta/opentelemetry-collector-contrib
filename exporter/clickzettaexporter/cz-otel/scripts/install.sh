#!/bin/bash
set -e

# cz-otel remote installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/clickzetta/opentelemetry-collector-contrib/main/exporter/clickzettaexporter/cz-otel/scripts/install.sh | bash
#
# Environment variables:
#   CZ_OTEL_VERSION  - Version to install (default: latest)
#   INSTALL_DIR      - Installation directory (default: ~/.cz-otel)

# --- Configuration -----------------------------------------------------------

REPO="clickzetta/opentelemetry-collector-contrib"
BINARY_NAME="cz-otel"
COLLECTOR_NAME="otelcol-clickzetta"
DEFAULT_INSTALL_DIR="$HOME/.cz-otel"
INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"

# --- Helper functions ---------------------------------------------------------

info() {
  printf "\033[1;34m==>\033[0m %s\n" "$1"
}

success() {
  printf "\033[1;32m==>\033[0m %s\n" "$1"
}

error() {
  printf "\033[1;31mError:\033[0m %s\n" "$1" >&2
  exit 1
}

# Detect OS
detect_os() {
  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux)  echo "linux" ;;
    darwin) echo "darwin" ;;
    mingw*|msys*|cygwin*) echo "windows" ;;
    *) error "Unsupported operating system: $os" ;;
  esac
}

# Detect architecture
detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) error "Unsupported architecture: $arch" ;;
  esac
}

# Get the latest release version from GitHub
get_latest_version() {
  local url="https://api.github.com/repos/${REPO}/releases/latest"
  local version

  if command -v curl &>/dev/null; then
    version=$(curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  elif command -v wget &>/dev/null; then
    version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  else
    error "Neither curl nor wget found. Please install one of them."
  fi

  # Strip leading 'v' if present
  echo "${version#v}"
}

# Download a file
download() {
  local url="$1"
  local dest="$2"

  info "Downloading $url"
  if command -v curl &>/dev/null; then
    curl -fsSL -o "$dest" "$url"
  elif command -v wget &>/dev/null; then
    wget -qO "$dest" "$url"
  else
    error "Neither curl nor wget found."
  fi
}

# --- Main logic ---------------------------------------------------------------

main() {
  local os arch version package_name download_url tmp_dir

  info "Detecting platform..."
  os="$(detect_os)"
  arch="$(detect_arch)"
  info "Platform: ${os}/${arch}"

  # Determine version
  if [ -n "${CZ_OTEL_VERSION:-}" ]; then
    version="$CZ_OTEL_VERSION"
    info "Using specified version: $version"
  else
    info "Fetching latest version..."
    version="$(get_latest_version)"
    if [ -z "$version" ]; then
      error "Failed to determine latest version. Set CZ_OTEL_VERSION manually."
    fi
    info "Latest version: $version"
  fi

  # Build download URL
  package_name="${BINARY_NAME}-${version}-${os}-${arch}.tar.gz"
  download_url="https://github.com/${REPO}/releases/download/v${version}/${package_name}"

  # Create temp directory for download
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT

  # Download the package
  download "$download_url" "${tmp_dir}/${package_name}"

  # Extract
  info "Extracting..."
  tar -xzf "${tmp_dir}/${package_name}" -C "$tmp_dir"

  # Find the extracted directory
  local extracted_dir
  extracted_dir=$(find "$tmp_dir" -maxdepth 1 -type d -name "${BINARY_NAME}-*" | head -1)
  if [ -z "$extracted_dir" ]; then
    # Binaries might be at top level in the tar
    extracted_dir="$tmp_dir"
  fi

  # Install
  info "Installing to ${INSTALL_DIR}..."
  mkdir -p "${INSTALL_DIR}/bin"
  mkdir -p "${INSTALL_DIR}/logs"

  # Copy binaries
  if [ -f "${extracted_dir}/bin/${BINARY_NAME}" ]; then
    cp "${extracted_dir}/bin/${BINARY_NAME}" "${INSTALL_DIR}/bin/${BINARY_NAME}"
  elif [ -f "${extracted_dir}/${BINARY_NAME}" ]; then
    cp "${extracted_dir}/${BINARY_NAME}" "${INSTALL_DIR}/bin/${BINARY_NAME}"
  fi

  if [ -f "${extracted_dir}/bin/${COLLECTOR_NAME}" ]; then
    cp "${extracted_dir}/bin/${COLLECTOR_NAME}" "${INSTALL_DIR}/bin/${COLLECTOR_NAME}"
  elif [ -f "${extracted_dir}/${COLLECTOR_NAME}" ]; then
    cp "${extracted_dir}/${COLLECTOR_NAME}" "${INSTALL_DIR}/bin/${COLLECTOR_NAME}"
  fi

  # Set executable permissions
  chmod +x "${INSTALL_DIR}/bin/${BINARY_NAME}" 2>/dev/null || true
  chmod +x "${INSTALL_DIR}/bin/${COLLECTOR_NAME}" 2>/dev/null || true

  # Verify installation
  if [ ! -f "${INSTALL_DIR}/bin/${BINARY_NAME}" ]; then
    error "Installation failed: ${BINARY_NAME} binary not found after extraction."
  fi

  success "Installation complete!"
  echo ""

  # PATH instructions
  if echo "$PATH" | tr ':' '\n' | grep -qx "${INSTALL_DIR}/bin"; then
    success "${INSTALL_DIR}/bin is already in your PATH."
  else
    echo "Add cz-otel to your PATH by adding the following to your shell profile:"
    echo ""
    echo "  For bash (~/.bashrc or ~/.bash_profile):"
    echo "    export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
    echo ""
    echo "  For zsh (~/.zshrc):"
    echo "    export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
    echo ""
    echo "Then reload your shell or run:"
    echo "    source ~/.bashrc   # or source ~/.zshrc"
    echo ""
  fi

  echo "Next steps:"
  echo "  1. Run 'cz-otel config init' to configure your ClickZetta connection"
  echo "  2. Run 'cz-otel start' to launch the collector"
  echo ""
}

main "$@"
