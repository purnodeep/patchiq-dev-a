#!/usr/bin/env bash
set -euo pipefail

# PatchIQ Agent Installer for Linux
# Usage: sudo ./install-agent-linux.sh --server HOST:PORT --token TOKEN [OPTIONS]

readonly SCRIPT_VERSION="1.0.0"
readonly DEFAULT_INSTALL_DIR="/usr/local/bin"
readonly DEFAULT_DATA_DIR="/var/lib/patchiq"
readonly DEFAULT_CONFIG_DIR="/etc/patchiq"
readonly SERVICE_NAME="patchiq-agent"
readonly BINARY_NAME="patchiq-agent"

# Exit codes
readonly EXIT_OK=0
readonly EXIT_ERROR=1
readonly EXIT_MISSING_PARAMS=2
readonly EXIT_DOWNLOAD_FAILED=3
readonly EXIT_CHECKSUM_MISMATCH=4

# Parameters
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
PatchIQ Agent Installer for Linux v${SCRIPT_VERSION}

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
        aarch64) echo "arm64" ;;
        *)       log_error "Unsupported architecture: ${arch}"; exit "${EXIT_ERROR}" ;;
    esac
}

download_binary() {
    local url="$1"
    local dest="$2"

    log_info "Downloading agent binary from ${url}"
    if command -v curl &>/dev/null; then
        curl -fsSL -o "${dest}" "${url}" || { log_error "Download failed from ${url}"; exit "${EXIT_DOWNLOAD_FAILED}"; }
    elif command -v wget &>/dev/null; then
        wget -q -O "${dest}" "${url}" || { log_error "Download failed from ${url}"; exit "${EXIT_DOWNLOAD_FAILED}"; }
    else
        log_error "Neither curl nor wget found. Install one and retry."
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
    if command -v sha256sum &>/dev/null; then
        actual="$(sha256sum "${file}" | awk '{print $1}')"
    elif command -v shasum &>/dev/null; then
        actual="$(shasum -a 256 "${file}" | awk '{print $1}')"
    else
        log_error "Cannot verify checksum: neither sha256sum nor shasum found"
        exit "${EXIT_ERROR}"
    fi

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
    if command -v curl &>/dev/null; then
        content="$(curl -fsSL "${checksum_url}" 2>/dev/null)" || return 1
    elif command -v wget &>/dev/null; then
        content="$(wget -q -O- "${checksum_url}" 2>/dev/null)" || return 1
    else
        return 1
    fi

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

        # Resolve checksum: explicit param > fetch from URL > skip
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

install_systemd_service() {
    local unit_src
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    unit_src="${script_dir}/../configs/agent/patchiq-agent.service"

    local unit_dest="/etc/systemd/system/${SERVICE_NAME}.service"

    # If bundled unit file exists, use it; otherwise generate inline
    if [[ -f "${unit_src}" ]]; then
        log_info "Installing systemd unit from ${unit_src}"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "cp ${unit_src} ${unit_dest}"
        else
            cp "${unit_src}" "${unit_dest}"
        fi
    else
        log_info "Generating systemd unit file"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "Write systemd unit to ${unit_dest}"
        else
            cat > "${unit_dest}" <<UNIT
[Unit]
Description=PatchIQ Agent
Documentation=https://github.com/skenzeriq/patchiq
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=always
RestartSec=5
Environment=PATCHIQ_AGENT_DATA_DIR=${DEFAULT_DATA_DIR}
WorkingDirectory=${DEFAULT_DATA_DIR}
LimitNOFILE=65536
# Hardening: prevent privilege escalation and restrict filesystem access
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${DEFAULT_DATA_DIR} ${DEFAULT_CONFIG_DIR}

[Install]
WantedBy=multi-user.target
UNIT
        fi
    fi

    log_info "Enabling and starting ${SERVICE_NAME} service"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "systemctl daemon-reload"
        log_dry "systemctl enable --now ${SERVICE_NAME}"
        return
    fi

    if ! systemctl daemon-reload; then
        log_error "systemctl daemon-reload failed"
        exit "${EXIT_ERROR}"
    fi
    if ! systemctl enable --now "${SERVICE_NAME}"; then
        log_error "Failed to enable/start ${SERVICE_NAME}. Check: journalctl -u ${SERVICE_NAME}"
        exit "${EXIT_ERROR}"
    fi
}

verify_install() {
    log_info "Verifying installation"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "systemctl is-active ${SERVICE_NAME}"
        return
    fi

    sleep 2
    if systemctl is-active --quiet "${SERVICE_NAME}"; then
        log_info "Service ${SERVICE_NAME} is running"
    else
        log_error "Service ${SERVICE_NAME} is not running. Check: journalctl -u ${SERVICE_NAME}"
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
    log_info "PatchIQ Agent Installer v${SCRIPT_VERSION}"
    log_info "Platform: linux/${arch}"
    log_info "Server: ${SERVER}"

    create_dirs
    install_binary
    enroll_agent
    install_systemd_service
    verify_install

    log_info "Installation complete"
}

main "$@"
