#!/usr/bin/env bash
set -euo pipefail

# PatchIQ Agent Installer for macOS
# Usage: sudo ./install-agent-macos.sh --server <url> --token <token> --binary <path>
# Uninstall: sudo ./install-agent-macos.sh --uninstall

LABEL="com.patchiq.agent"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="patchiq-agent"
PLIST_PATH="/Library/LaunchDaemons/${LABEL}.plist"
DATA_DIR="/var/lib/patchiq"
LOG_DIR="/var/log"

usage() {
    echo "Usage: sudo $0 --server <url> --token <token> --binary <path>"
    echo "       sudo $0 --uninstall"
    exit 1
}

require_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "Error: this script must be run as root (use sudo)" >&2
        exit 1
    fi
}

install_agent() {
    local server_url="$1"
    local token="$2"
    local binary_path="$3"

    echo "==> Installing PatchIQ Agent..."

    if [[ -z "$binary_path" ]]; then
        echo "Error: --binary <path> is required" >&2
        exit 1
    fi

    if [[ ! -f "$binary_path" ]]; then
        echo "Error: binary not found at $binary_path" >&2
        exit 1
    fi

    cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
    chown root:wheel "${INSTALL_DIR}/${BINARY_NAME}"
    echo "    Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"

    mkdir -p "$DATA_DIR"
    echo "    Data directory: $DATA_DIR"

    cat > "$PLIST_PATH" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BINARY_NAME}</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATCHIQ_AGENT_SERVER_ADDRESS</key>
        <string>${server_url}</string>
        <key>PATCHIQ_AGENT_ENROLLMENT_TOKEN</key>
        <string>${token}</string>
        <key>PATCHIQ_AGENT_DATA_DIR</key>
        <string>${DATA_DIR}</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${LOG_DIR}/patchiq-agent.log</string>
    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/patchiq-agent.err</string>
</dict>
</plist>
PLIST

    chmod 644 "$PLIST_PATH"
    chown root:wheel "$PLIST_PATH"
    echo "    Plist installed to $PLIST_PATH"

    launchctl load "$PLIST_PATH"
    echo "    Daemon loaded"

    if launchctl list | grep -q "$LABEL"; then
        echo "==> PatchIQ Agent installed and running."
    else
        echo "Warning: daemon may not have started. Check: launchctl list | grep $LABEL" >&2
    fi
}

uninstall_agent() {
    echo "==> Uninstalling PatchIQ Agent..."

    if [[ -f "$PLIST_PATH" ]]; then
        launchctl unload "$PLIST_PATH" 2>/dev/null || true
        rm -f "$PLIST_PATH"
        echo "    Plist removed"
    fi

    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        echo "    Binary removed"
    fi

    echo "    Data directory $DATA_DIR left intact (remove manually if needed)"
    echo "==> PatchIQ Agent uninstalled."
}

# Parse arguments
SERVER_URL=""
TOKEN=""
BINARY_PATH=""
UNINSTALL=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --server)
            SERVER_URL="$2"
            shift 2
            ;;
        --token)
            TOKEN="$2"
            shift 2
            ;;
        --binary)
            BINARY_PATH="$2"
            shift 2
            ;;
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        *)
            usage
            ;;
    esac
done

require_root

if $UNINSTALL; then
    uninstall_agent
    exit 0
fi

if [[ -z "$SERVER_URL" || -z "$TOKEN" ]]; then
    echo "Error: --server and --token are required" >&2
    usage
fi

install_agent "$SERVER_URL" "$TOKEN" "$BINARY_PATH"
