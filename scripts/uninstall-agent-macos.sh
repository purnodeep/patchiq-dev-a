#!/usr/bin/env bash
set -euo pipefail

# PatchIQ Agent Uninstaller for macOS
# Usage: sudo ./uninstall-agent-macos.sh [--remove-data]

readonly SCRIPT_VERSION="1.0.0"
readonly DEFAULT_INSTALL_DIR="/usr/local/bin"
readonly DEFAULT_DATA_DIR="/var/lib/patchiq"
readonly DEFAULT_CONFIG_DIR="/etc/patchiq"
readonly LAUNCHD_LABEL="com.patchiq.agent"
readonly PLIST_PATH="/Library/LaunchDaemons/${LAUNCHD_LABEL}.plist"
readonly BINARY_NAME="patchiq-agent"

REMOVE_DATA=false
DRY_RUN=false

log_info()  { echo "[INFO]  $*"; }
log_error() { echo "[ERROR] $*" >&2; }
log_warn()  { echo "[WARN]  $*" >&2; }
log_dry()   { echo "[DRY-RUN] $*"; }

usage() {
    cat <<EOF
PatchIQ Agent Uninstaller for macOS v${SCRIPT_VERSION}

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
    if [[ ! -f "${PLIST_PATH}" ]]; then
        log_info "Launchd plist not found, skipping service stop"
        return
    fi

    log_info "Unloading launchd daemon ${LAUNCHD_LABEL}"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "launchctl bootout system ${PLIST_PATH}  # (or unload on macOS < 13)"
        log_dry "rm ${PLIST_PATH}"
        return
    fi

    # launchctl unload is deprecated since macOS 10.10 and removed in Ventura (13.0).
    local macos_version
    macos_version="$(sw_vers -productVersion 2>/dev/null || echo "0")"
    local major_version="${macos_version%%.*}"
    if [[ "${major_version}" -ge 13 ]]; then
        launchctl bootout system "${PLIST_PATH}" 2>/dev/null || true
    else
        launchctl unload "${PLIST_PATH}" 2>/dev/null || true
    fi
    rm -f "${PLIST_PATH}"
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

remove_logs() {
    log_info "Removing log files"
    if [[ "${DRY_RUN}" == true ]]; then
        log_dry "rm -f /var/log/patchiq-agent.log /var/log/patchiq-agent.err"
        return
    fi
    rm -f /var/log/patchiq-agent.log /var/log/patchiq-agent.err
}

main() {
    parse_args "$@"

    if [[ "${DRY_RUN}" != true ]]; then
        require_root
    fi

    log_info "PatchIQ Agent Uninstaller for macOS v${SCRIPT_VERSION}"

    stop_service
    remove_binary
    remove_config
    remove_data
    remove_logs

    log_info "Uninstall complete"
}

main "$@"
