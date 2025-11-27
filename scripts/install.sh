#!/usr/bin/env bash
# Kilt Installer Script
# One-line installation and bootstrapping for Kilt
#
# Usage:
#   curl -sL https://get.kilt.pro | bash -s -- [repo-url]
#   curl -sL https://get.kilt.pro | bash -s -- https://github.com/user/dotfiles

set -euo pipefail

# Colors for output (respect NO_COLOR env var)
if [[ -t 1 ]] && [[ "${NO_COLOR:-}" == "" ]]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  BLUE='\033[0;34m'
  NC='\033[0m' # No Color
else
  RED=''
  GREEN=''
  YELLOW=''
  BLUE=''
  NC=''
fi

# Configuration
KILT_VERSION="${KILT_VERSION:-latest}"
KILT_REPO="${KILT_REPO:-unravelling/kilt}"
GITHUB_API="https://api.github.com"
GITHUB_RELEASES="https://github.com/${KILT_REPO}/releases"
INSTALL_DIR="${KILT_INSTALL_DIR:-}"
TEMP_DIR=""

# Error handling
cleanup() {
  local exit_code=$?
  if [[ -n "${TEMP_DIR}" ]] && [[ -d "${TEMP_DIR}" ]]; then
    rm -rf "${TEMP_DIR}"
  fi
  if [[ $exit_code -ne 0 ]]; then
    echo -e "${RED}Installation failed.${NC}" >&2
  fi
  exit $exit_code
}

trap cleanup EXIT INT TERM

# Logging functions
log_info() {
  echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
  echo -e "${GREEN}✓${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}⚠${NC} $*"
}

log_error() {
  echo -e "${RED}✗${NC} $*" >&2
}

# Detect OS and architecture
detect_os() {
  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  
  case "$os" in
    darwin)
      echo "darwin"
      ;;
    linux)
      echo "linux"
      ;;
    *)
      log_error "Unsupported OS: $os"
      exit 1
      ;;
  esac
}

detect_arch() {
  local arch
  arch="$(uname -m | tr '[:upper:]' '[:lower:]')"
  
  case "$arch" in
    x86_64|amd64)
      echo "amd64"
      ;;
    arm64|aarch64)
      echo "arm64"
      ;;
    *)
      log_error "Unsupported architecture: $arch"
      exit 1
      ;;
  esac
}

# Get latest release version from GitHub API
get_latest_version() {
  local version
  if command -v curl >/dev/null 2>&1; then
    version=$(curl -sSL "${GITHUB_API}/repos/${KILT_REPO}/releases/latest" | \
      grep -oP '"tag_name":\s*"\K[^"]+' | head -1 || echo "")
  elif command -v wget >/dev/null 2>&1; then
    version=$(wget -qO- "${GITHUB_API}/repos/${KILT_REPO}/releases/latest" | \
      grep -oP '"tag_name":\s*"\K[^"]+' | head -1 || echo "")
  fi
  
  if [[ -z "$version" ]]; then
    log_warn "Could not fetch latest version, using 'latest'"
    echo "latest"
  else
    echo "$version"
  fi
}

# Determine install directory
get_install_dir() {
  if [[ -n "${INSTALL_DIR}" ]]; then
    echo "${INSTALL_DIR}"
    return
  fi
  
  # Prefer /usr/local/bin if writable, otherwise ~/.local/bin
  if [[ -w /usr/local/bin ]] || sudo -n true 2>/dev/null; then
    echo "/usr/local/bin"
  else
    echo "${HOME}/.local/bin"
  fi
}

# Download binary from GitHub releases
# Tries standalone binary first, then falls back to archive extraction
download_binary() {
  local os=$1
  local arch=$2
  local version=$3
  local output_file=$4
  
  local binary_name="kilt-${os}-${arch}"
  local archive_name="kilt-${version}-${os}-${arch}.tar.gz"
  local download_url
  local archive_file="${TEMP_DIR}/${archive_name}"
  
  if [[ "$version" == "latest" ]]; then
    version=$(get_latest_version)
  fi
  
  # Remove 'v' prefix if present
  version="${version#v}"
  
  log_info "Downloading Kilt ${version} for ${os}/${arch}..."
  
  # Try standalone binary first (for backward compatibility)
  download_url="${GITHUB_RELEASES}/download/v${version}/${binary_name}"
  log_info "Trying standalone binary: ${download_url}"
  
  if command -v curl >/dev/null 2>&1; then
    if curl -fsSL -o "${output_file}" "${download_url}" 2>/dev/null; then
      log_success "Downloaded standalone binary"
      return 0
    fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -qO "${output_file}" "${download_url}" 2>/dev/null; then
      log_success "Downloaded standalone binary"
      return 0
    fi
  fi
  
  # Fall back to archive download and extraction
  log_info "Standalone binary not found, trying archive..."
  download_url="${GITHUB_RELEASES}/download/v${version}/${archive_name}"
  log_info "URL: ${download_url}"
  
  if command -v curl >/dev/null 2>&1; then
    if ! curl -fsSL -o "${archive_file}" "${download_url}"; then
      log_error "Failed to download archive from ${download_url}"
      return 1
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -qO "${archive_file}" "${download_url}"; then
      log_error "Failed to download archive from ${download_url}"
      return 1
    fi
  else
    log_error "Neither curl nor wget is available. Please install one of them."
    return 1
  fi
  
  # Extract binary from archive
  log_info "Extracting binary from archive..."
  local extract_dir="${TEMP_DIR}/extract"
  mkdir -p "${extract_dir}"
  
  if ! tar -xzf "${archive_file}" -C "${extract_dir}" 2>/dev/null; then
    log_error "Failed to extract archive"
    return 1
  fi
  
  # Find the binary in the extracted files (look for "kilt" or binary_name)
  local extracted_binary
  extracted_binary=$(find "${extract_dir}" -name "kilt" -type f | head -1)
  
  # If not found, try the expected binary name
  if [[ -z "${extracted_binary}" ]]; then
    extracted_binary=$(find "${extract_dir}" -name "${binary_name}" -type f | head -1)
  fi
  
  if [[ -z "${extracted_binary}" ]] || [[ ! -f "${extracted_binary}" ]]; then
    log_error "Binary not found in archive"
    return 1
  fi
  
  # Copy extracted binary to output location
  cp "${extracted_binary}" "${output_file}"
  chmod +x "${output_file}"
  
  log_success "Extracted binary from archive"
}

# Verify checksum (if checksums file is available)
verify_checksum() {
  local binary_file=$1
  local os=$2
  local arch=$3
  local version=$4
  
  local binary_name="kilt-${os}-${arch}"
  local checksums_url="${GITHUB_RELEASES}/download/v${version}/checksums.txt"
  local checksums_file="${TEMP_DIR}/checksums.txt"
  
  # Remove 'v' prefix if present
  version="${version#v}"
  
  log_info "Verifying checksum..."
  
  # Download checksums file
  if command -v curl >/dev/null 2>&1; then
    if ! curl -fsSL -o "${checksums_file}" "${checksums_url}" 2>/dev/null; then
      log_warn "Checksums file not available, skipping verification"
      return 0
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -qO "${checksums_file}" "${checksums_url}" 2>/dev/null; then
      log_warn "Checksums file not available, skipping verification"
      return 0
    fi
  else
    log_warn "Cannot download checksums file, skipping verification"
    return 0
  fi
  
  # Calculate SHA256 of downloaded binary
  local calculated_checksum
  if command -v shasum >/dev/null 2>&1; then
    calculated_checksum=$(shasum -a 256 "${binary_file}" | cut -d' ' -f1)
  elif command -v sha256sum >/dev/null 2>&1; then
    calculated_checksum=$(sha256sum "${binary_file}" | cut -d' ' -f1)
  else
    log_warn "No SHA256 tool available, skipping verification"
    return 0
  fi
  
  # Find expected checksum in checksums file
  local expected_checksum
  expected_checksum=$(grep "${binary_name}" "${checksums_file}" | cut -d' ' -f1 || echo "")
  
  if [[ -z "${expected_checksum}" ]]; then
    log_warn "Checksum for ${binary_name} not found in checksums file, skipping verification"
    return 0
  fi
  
  if [[ "${calculated_checksum}" != "${expected_checksum}" ]]; then
    log_error "Checksum verification failed!"
    log_error "Expected: ${expected_checksum}"
    log_error "Got:      ${calculated_checksum}"
    return 1
  fi
  
  log_success "Checksum verified"
  return 0
}

# Install binary to target directory
install_binary() {
  local binary_file=$1
  local install_dir=$2
  
  local target="${install_dir}/kilt"
  
  # Create install directory if it doesn't exist
  if [[ ! -d "${install_dir}" ]]; then
    log_info "Creating directory: ${install_dir}"
    if [[ "${install_dir}" == "/usr/local/bin" ]] && ! sudo -n true 2>/dev/null; then
      sudo mkdir -p "${install_dir}"
    else
      mkdir -p "${install_dir}"
    fi
  fi
  
  # Make binary executable
  chmod +x "${binary_file}"
  
  # Install binary
  log_info "Installing to ${target}..."
  if [[ "${install_dir}" == "/usr/local/bin" ]] && ! sudo -n true 2>/dev/null; then
    sudo cp "${binary_file}" "${target}"
    sudo chmod +x "${target}"
  else
    cp "${binary_file}" "${target}"
    chmod +x "${target}"
  fi
  
  log_success "Installed Kilt to ${target}"
  
  # Add to PATH if ~/.local/bin
  if [[ "${install_dir}" == "${HOME}/.local/bin" ]]; then
    local shell_rc=""
    if [[ -n "${ZSH_VERSION:-}" ]]; then
      shell_rc="${HOME}/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]]; then
      shell_rc="${HOME}/.bashrc"
    fi
    
    if [[ -n "${shell_rc}" ]] && ! grep -q "${install_dir}" "${shell_rc}" 2>/dev/null; then
      log_info "Adding ${install_dir} to PATH in ${shell_rc}"
      echo "" >> "${shell_rc}"
      echo "# Added by Kilt installer" >> "${shell_rc}"
      echo "export PATH=\"\${PATH}:${install_dir}\"" >> "${shell_rc}"
      log_success "Added to PATH. Run 'source ${shell_rc}' or restart your terminal."
    fi
  fi
}

# Bootstrap dotfiles repository
bootstrap_repo() {
  local repo_url=$1
  
  if [[ -z "${repo_url}" ]]; then
    return 0
  fi
  
  log_info "Bootstrapping dotfiles repository: ${repo_url}"
  
  # Check if kilt is available
  local kilt_cmd
  kilt_cmd=$(command -v kilt || echo "")
  
  if [[ -z "${kilt_cmd}" ]]; then
    # Try to find it in common locations
    if [[ -x "/usr/local/bin/kilt" ]]; then
      kilt_cmd="/usr/local/bin/kilt"
    elif [[ -x "${HOME}/.local/bin/kilt" ]]; then
      kilt_cmd="${HOME}/.local/bin/kilt"
    else
      log_warn "Kilt binary not found in PATH. Please run 'kilt init ${repo_url}' manually."
      return 1
    fi
  fi
  
  log_info "Initializing Kilt with repository..."
  if "${kilt_cmd}" init "${repo_url}"; then
    log_success "Repository initialized successfully"
    log_info "Run 'kilt sync' to apply your configuration"
  else
    log_error "Failed to initialize repository"
    return 1
  fi
}

# Main installation function
main() {
  local repo_url="${1:-}"
  
  log_info "Kilt Installer"
  log_info "=============="
  
  # Detect OS and architecture
  local os
  local arch
  os=$(detect_os)
  arch=$(detect_arch)
  
  log_info "Detected OS: ${os}, Architecture: ${arch}"
  
  # Get version
  local version="${KILT_VERSION}"
  if [[ "${version}" == "latest" ]]; then
    version=$(get_latest_version)
  fi
  
  log_info "Installing version: ${version}"
  
  # Create temporary directory
  TEMP_DIR=$(mktemp -d)
  local binary_file="${TEMP_DIR}/kilt"
  
  # Download binary
  if ! download_binary "${os}" "${arch}" "${version}" "${binary_file}"; then
    exit 1
  fi
  
  # Verify checksum
  if ! verify_checksum "${binary_file}" "${os}" "${arch}" "${version}"; then
    exit 1
  fi
  
  # Determine install directory
  local install_dir
  install_dir=$(get_install_dir)
  log_info "Install directory: ${install_dir}"
  
  # Install binary
  if ! install_binary "${binary_file}" "${install_dir}"; then
    exit 1
  fi
  
  # Verify installation
  local installed_kilt
  installed_kilt=$(command -v kilt || echo "${install_dir}/kilt")
  if [[ -x "${installed_kilt}" ]]; then
    local installed_version
    installed_version=$("${installed_kilt}" version 2>/dev/null || echo "unknown")
    log_success "Installation complete!"
    log_info "Kilt version: ${installed_version}"
    log_info "Location: ${installed_kilt}"
  else
    log_warn "Installation may have failed. Please verify manually."
  fi
  
  # Bootstrap repository if URL provided
  if [[ -n "${repo_url}" ]]; then
    bootstrap_repo "${repo_url}"
  else
    log_info "No repository URL provided. Run 'kilt init <repo-url>' to get started."
  fi
}

# Run main function
main "$@"

