#!/usr/bin/env bash
set -euo pipefail

# PatchIQ Agent Installer for macOS
# Usage: sudo ./install-agent-macos.sh --server HOST:PORT --token TOKEN [OPTIONS]

readonly SCRIPT_VERSION="1.0.0"
readonly DEFAULT_INSTALL_DIR="/usr/local/bin"
readonly DEFAULT_DATA_DIR="/var/lib/patchiq"
readonly DEFAULT_CONFIG_DIR="/etc/patchiq"
readonly LAUNCHD_LABEL="com.patchiq.agent"
readonly PLIST_PATH="/Library/LaunchDaemons/${LAUNCHD_LABEL}.plist"
readonly BINARY_NAME="patchiq-agent"

# Exit codes
readonly EXIT_OK=0
readonly EXIT_ERROR=1
readonly EXIT_MISSING_PARAMS=2
readonly EXIT_DOWNLOAD_FAILED=3
readonly EXIT_CHECKSUM_MISMATCH=4

SERVER=""
TOKEN=""
DOWNLOAD_URL=""
CHECKSUM=""
INSTALL_DIR="${DEFAULT_INSTALL_DIR}"
DRY_RUN=false

log_info()  { echo "[INFO]  $*"; }
log_error() { echo "[ERROR] $*" >&2; }
log_warn()  { echo "[WARN]  $*" >&2; }
log_dry()   { echo "[DRY-RUN] $*"; }

usage() {
    cat <<EOF
PatchIQ Agent Installer for macOS v${SCRIPT_VERSION}

Usage: sudo $0 --server HOST:PORT --token TOKEN [OPTIONS]

Required:
  --server URL        Patch Manager gRPC address (host:port)
  --token TOKEN       One-time enrollment token

Optional:
  --download-url URL  URL to download the agent binary
  --checksum SHA256   Expected SHA256 hex digest of the binary
  --install-dir DIR   Install directory (default: ${DEFAULT_INSTALL_DIR})
  --dry-run           Print actions without executing

Exit codes:
  0  Success
  1  General error
  2  Missing required parameters
  3  Download failed
  4  Checksum mismatch
EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --server)
                if [[ $# -lt 2 ]]; then
                    log_error "--server requires a value"
                    exit "${EXIT_MISSING_PARAMS}"
                fi
                SERVER="$2"; shift 2 ;;
            --token)
                if [[ $# -lt 2 ]]; then
                    log_error "--token requires a value"
                    exit "${EXIT_MISSING_PARAMS}"
                fi
                TOKEN="$2"; shift 2 ;;
            --download-url)
                if [[ $# -lt 2 ]]; then
                    log_error "--download-url requires a value"
                    exit "${EXIT_MISSING_PARAMS}"
                fi
                DOWNLOAD_URL="$2"; shift 2 ;;
            --checksum)
                if [[ $# -lt 2 ]]; then
                    log_error "--checksum requires a value"
                    exit "${EXIT_MISSING_PARAMS}"
                fi
                CHECKSUM="$2"; shift 2 ;;
            --install-dir)
                if [[ $# -lt 2 ]]; then
                    log_error "--install-dir requires a value"
                    exit "${EXIT_MISSING_PARAMS}"
                fi
                INSTALL_DIR="$2"; shift 2 ;;
            --dry-run)   DRY_RUN=true; shift ;;
            --help|-h)   usage; exit "${EXIT_OK}" ;;
            *)           log_error "Unknown option: $1"; usage; exit "${EXIT_MISSING_PARAMS}" ;;
        esac
    done
}

require_root() {
    if [[ "$(id -u)" -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit "${EXIT_ERROR}"
    fi
}

validate_params() {
    if [[ -z "${SERVER}" ]]; then
        log_error "Missing required parameter: --server"
        usage
        exit "${EXIT_MISSING_PARAMS}"
    fi
    if [[ -z "${TOKEN}" ]]; then
        log_error "Missing required parameter: --token"
        usage
        exit "${EXIT_MISSING_PARAMS}"
    fi

    # Validate server format: host:port
    if ! [[ "${SERVER}" =~ ^[a-zA-Z0-9._-]+:[0-9]+$ ]]; then
        log_error "Invalid --server format '${SERVER}': expected host:port (e.g. pm.example.com:50051)"
        exit "${EXIT_MISSING_PARAMS}"
    fi

    # Validate download URL uses HTTPS if provided
    if [[ -n "${DOWNLOAD_URL}" ]] && ! [[ "${DOWNLOAD_URL}" =~ ^https:// ]]; then
        log_error "Refusing to download over insecure connection: --download-url must use https://"
        exit "${EXIT_ERROR}"
    fi
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64)  echo "amd64" ;;
        arm64)   echo "arm64" ;;
        *)       log_error "Unsupported architecture: ${arch}"; exit "${EXIT_ERROR}" ;;
    esac
}

download_binary() {
    local url="$1"
    local dest="$2"

    log_info "Downloading agent binary from ${url}"
    if ! curl -fsSL -o "${dest}" "${url}"; then
        log_error "Download failed from ${url}"
        exit "${EXIT_DOWNLOAD_FAILED}"
    fi
}

verify_checksum() {
    local file="$1"
    local expected="$2"

    if [[ -z "${expected}" ]]; then
        log_warn "No checksum provided, skipping verification"
        return 0
    fi

    local actual
    actual="$(shasum -a 256 "${file}" | awk '{print $1}')"

    if [[ "${actual}" != "${expected}" ]]; then
        log_error "Checksum mismatch: expected ${expected}, got ${actual}"
        exit "${EXIT_CHECKSUM_MISMATCH}"
    fi
    log_info "Checksum verified: ${actual}"
}

fetch_checksum() {
    local url="$1"
    local checksum_url="${url}.sha256"

    log_info "Fetching checksum from ${checksum_url}"
    local content
    content="$(curl -fsSL "${checksum_url}" 2>/dev/null)" || return 1
    echo "${content}" | awk '{print $1}'
}

create_dirs() {
    log_info "Creating directories"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "mkdir -p ${INSTALL_DIR} ${DEFAULT_DATA_DIR} ${DEFAULT_CONFIG_DIR}"
        return
    fi
    mkdir -p "${INSTALL_DIR}" "${DEFAULT_DATA_DIR}" "${DEFAULT_CONFIG_DIR}"
}

install_binary() {
    local binary_path="${INSTALL_DIR}/${BINARY_NAME}"

    if [[ -n "${DOWNLOAD_URL}" ]]; then
        local tmp_file
        tmp_file="$(mktemp)"
        trap 'rm -f "${tmp_file}"' EXIT

        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "Download ${DOWNLOAD_URL} -> ${tmp_file}"
            log_dry "Verify checksum"
            log_dry "Move ${tmp_file} -> ${binary_path}"
            return
        fi

        download_binary "${DOWNLOAD_URL}" "${tmp_file}"

        local checksum_to_verify="${CHECKSUM}"
        if [[ -z "${checksum_to_verify}" ]]; then
            if ! checksum_to_verify="$(fetch_checksum "${DOWNLOAD_URL}")"; then
                checksum_to_verify=""
                log_warn "Failed to fetch checksum from ${DOWNLOAD_URL}.sha256 — installing WITHOUT integrity verification"
            fi
        fi
        verify_checksum "${tmp_file}" "${checksum_to_verify}"

        mv "${tmp_file}" "${binary_path}"
        chmod +x "${binary_path}"
        log_info "Binary installed to ${binary_path}"
    else
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "Verify binary exists at ${binary_path}"
            log_dry "chmod +x ${binary_path}"
            return
        fi
        if [[ ! -f "${binary_path}" ]]; then
            log_error "Binary not found at ${binary_path} and no --download-url provided"
            exit "${EXIT_ERROR}"
        fi
        log_info "Using existing binary at ${binary_path}"
        chmod +x "${binary_path}"
    fi
}

enroll_agent() {
    local binary_path="${INSTALL_DIR}/${BINARY_NAME}"

    log_info "Enrolling agent with server ${SERVER}"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "${binary_path} install --server ${SERVER} --token [REDACTED] --non-interactive"
        return
    fi

    # Pass token via env var to avoid exposing it in the process list (ps/top).
    if ! PATCHIQ_ENROLLMENT_TOKEN="${TOKEN}" "${binary_path}" install --server "${SERVER}" --non-interactive; then
        log_error "Agent enrollment failed. Verify: (1) server ${SERVER} is reachable, (2) token is valid"
        exit "${EXIT_ERROR}"
    fi
}

install_launchd_service() {
    local plist_src
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    plist_src="${script_dir}/../configs/agent/com.patchiq.agent.plist"

    if [[ -f "${plist_src}" ]]; then
        log_info "Installing launchd plist from ${plist_src}"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "cp ${plist_src} ${PLIST_PATH}"
        else
            cp "${plist_src}" "${PLIST_PATH}"
        fi
    else
        log_info "Generating launchd plist"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "Write plist to ${PLIST_PATH}"
        else
            cat > "${PLIST_PATH}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>${LAUNCHD_LABEL}</string>
	<key>ProgramArguments</key>
	<array>
		<string>${INSTALL_DIR}/${BINARY_NAME}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/patchiq-agent.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/patchiq-agent.err</string>
</dict>
</plist>
PLIST
        fi
    fi

    log_info "Loading launchd daemon"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "launchctl bootstrap system ${PLIST_PATH}"
        return
    fi

    # launchctl load is deprecated since macOS 10.10 and removed in Ventura (13.0).
    # Use launchctl bootstrap for macOS 13+ and fall back to load for older versions.
    local macos_version
    macos_version="$(sw_vers -productVersion 2>/dev/null || echo "0")"
    local major_version="${macos_version%%.*}"
    if [[ "${major_version}" -ge 13 ]]; then
        launchctl bootstrap system "${PLIST_PATH}"
    else
        launchctl load "${PLIST_PATH}"
    fi
}

verify_install() {
    log_info "Verifying installation"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "launchctl list | grep ${LAUNCHD_LABEL}"
        return
    fi

    sleep 2
    if launchctl list | grep -q "${LAUNCHD_LABEL}"; then
        log_info "Daemon ${LAUNCHD_LABEL} is loaded"
    else
        log_error "Daemon ${LAUNCHD_LABEL} is not loaded. Check: /var/log/patchiq-agent.err"
        exit "${EXIT_ERROR}"
    fi
}

main() {
    parse_args "$@"

    validate_params

    if [[ "${DRY_RUN}" != true ]]; then
        require_root
    fi

    local arch
    arch="$(detect_arch)"
    log_info "PatchIQ Agent Installer for macOS v${SCRIPT_VERSION}"
    log_info "Platform: darwin/${arch}"
    log_info "Server: ${SERVER}"

    create_dirs
    install_binary
    enroll_agent
    install_launchd_service
    verify_install

    log_info "Installation complete"
}

main "$@"
