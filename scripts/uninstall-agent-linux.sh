#!/usr/bin/env bash
set -euo pipefail

# PatchIQ Agent Uninstaller for Linux
# Usage: sudo ./uninstall-agent-linux.sh [--remove-data]

readonly SCRIPT_VERSION="1.0.0"
readonly DEFAULT_INSTALL_DIR="/usr/local/bin"
readonly DEFAULT_DATA_DIR="/var/lib/patchiq"
readonly DEFAULT_CONFIG_DIR="/etc/patchiq"
readonly SERVICE_NAME="patchiq-agent"
readonly BINARY_NAME="patchiq-agent"

REMOVE_DATA=false
DRY_RUN=false

log_info()  { echo "[INFO]  $*"; }
log_error() { echo "[ERROR] $*" >&2; }
log_warn()  { echo "[WARN]  $*" >&2; }
log_dry()   { echo "[DRY-RUN] $*"; }

usage() {
    cat <<EOF
PatchIQ Agent Uninstaller for Linux v${SCRIPT_VERSION}

Usage: sudo $0 [OPTIONS]

Options:
  --remove-data   Also remove agent data directory (${DEFAULT_DATA_DIR})
  --dry-run       Print actions without executing
  --help, -h      Show this help
EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --remove-data) REMOVE_DATA=true; shift ;;
            --dry-run)     DRY_RUN=true; shift ;;
            --help|-h)     usage; exit 0 ;;
            *)             log_error "Unknown option: $1"; usage; exit 1 ;;
        esac
    done
}

require_root() {
    if [[ "$(id -u)" -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

stop_service() {
    local unit_file="/etc/systemd/system/${SERVICE_NAME}.service"
    if [[ ! -f "${unit_file}" ]]; then
        log_info "Systemd unit not found, skipping service stop"
        return
    fi

    log_info "Stopping and disabling ${SERVICE_NAME} service"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "systemctl stop ${SERVICE_NAME}"
        log_dry "systemctl disable ${SERVICE_NAME}"
        log_dry "rm ${unit_file}"
        log_dry "systemctl daemon-reload"
        return
    fi

    systemctl stop "${SERVICE_NAME}" 2>/dev/null || true
    systemctl disable "${SERVICE_NAME}" 2>/dev/null || true
    rm -f "${unit_file}"
    systemctl daemon-reload
}

remove_binary() {
    local binary_path="${DEFAULT_INSTALL_DIR}/${BINARY_NAME}"
    if [[ -f "${binary_path}" ]]; then
        log_info "Removing binary: ${binary_path}"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "rm ${binary_path}"
        else
            rm -f "${binary_path}"
        fi
    else
        log_info "Binary not found at ${binary_path}, skipping"
    fi
}

remove_config() {
    if [[ -d "${DEFAULT_CONFIG_DIR}" ]]; then
        log_info "Removing config directory: ${DEFAULT_CONFIG_DIR}"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "rm -rf ${DEFAULT_CONFIG_DIR}"
        else
            rm -rf "${DEFAULT_CONFIG_DIR}"
        fi
    fi
}

remove_data() {
    if [[ "${REMOVE_DATA}" != true ]]; then
        log_info "Keeping data directory: ${DEFAULT_DATA_DIR} (use --remove-data to delete)"
        return
    fi

    if [[ -d "${DEFAULT_DATA_DIR}" ]]; then
        log_info "Removing data directory: ${DEFAULT_DATA_DIR}"
        if [[ "${DRY_RUN}" == true ]]; then
            log_dry "rm -rf ${DEFAULT_DATA_DIR}"
        else
            rm -rf "${DEFAULT_DATA_DIR}"
        fi
    fi
}

main() {
    parse_args "$@"

    if [[ "${DRY_RUN}" != true ]]; then
        require_root
    fi

    log_info "PatchIQ Agent Uninstaller v${SCRIPT_VERSION}"

    stop_service
    remove_binary
    remove_config
    remove_data

    log_info "Uninstall complete"
}

main "$@"
