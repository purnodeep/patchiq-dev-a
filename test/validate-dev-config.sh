#!/usr/bin/env bash
# Validates dev config consistency across all sources.
# Exits non-zero if any check fails.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILURES=0

fail() {
    echo "FAIL: $1"
    FAILURES=$((FAILURES + 1))
}

pass() {
    echo "ok:   $1"
}

check_contains() {
    local file="$1"
    local pattern="$2"
    local description="$3"
    if grep -q "$pattern" "$file" 2>/dev/null; then
        pass "$description"
    else
        fail "$description (pattern '$pattern' not found in $file)"
    fi
}

check_not_contains() {
    local file="$1"
    local pattern="$2"
    local description="$3"
    if ! grep -q "$pattern" "$file" 2>/dev/null; then
        pass "$description"
    else
        fail "$description (forbidden pattern '$pattern' found in $file)"
    fi
}

check_file_exists() {
    local file="$1"
    local description="$2"
    if [ -f "$file" ]; then
        pass "$description"
    else
        fail "$description (file not found: $file)"
    fi
}

echo "=== Dev Config Consistency Validation ==="
echo ""

# --- DB name checks ---
echo "-- DB name checks --"

# docker-compose.dev.yml: POSTGRES_DB must be patchiq (not patchiq_dev)
check_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'POSTGRES_DB=patchiq$\|POSTGRES_DB: patchiq$\|POSTGRES_DB=patchiq[^_]' \
    "docker-compose.dev.yml: POSTGRES_DB=patchiq"
check_not_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'POSTGRES_DB=patchiq_dev\|POSTGRES_DB: patchiq_dev' \
    "docker-compose.dev.yml: POSTGRES_DB must not be patchiq_dev"

# configs/server.yaml: database.url must contain /patchiq?
check_contains \
    "$REPO_ROOT/configs/server.yaml" \
    '/patchiq?' \
    "configs/server.yaml: database.url contains /patchiq?"

# configs/hub.yaml: database.url must contain /patchiq_hub?
check_contains \
    "$REPO_ROOT/configs/hub.yaml" \
    '/patchiq_hub?' \
    "configs/hub.yaml: database.url contains /patchiq_hub?"

# .env.example: DATABASE_URL must contain /patchiq?
check_contains \
    "$REPO_ROOT/.env.example" \
    '/patchiq?' \
    ".env.example: DATABASE_URL contains /patchiq?"
check_not_contains \
    "$REPO_ROOT/.env.example" \
    '/patchiq_dev?' \
    ".env.example: DATABASE_URL must not contain /patchiq_dev?"

# docker-compose.dev.yml: healthcheck must use -d patchiq (not patchiq_dev)
check_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'pg_isready.*-d patchiq' \
    "docker-compose.dev.yml: pg_isready healthcheck uses -d patchiq"
check_not_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'pg_isready.*patchiq_dev' \
    "docker-compose.dev.yml: pg_isready healthcheck must not use patchiq_dev"

echo ""

# --- Password checks ---
echo "-- Password checks --"

# docker-compose.dev.yml: POSTGRES_PASSWORD must be patchiq (not patchiq_dev)
check_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'POSTGRES_PASSWORD=patchiq$\|POSTGRES_PASSWORD: patchiq$\|POSTGRES_PASSWORD=patchiq[^_]' \
    "docker-compose.dev.yml: POSTGRES_PASSWORD=patchiq"
check_not_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'POSTGRES_PASSWORD=patchiq_dev\|POSTGRES_PASSWORD: patchiq_dev' \
    "docker-compose.dev.yml: POSTGRES_PASSWORD must not be patchiq_dev"

# configs/server.yaml: password in URL must be patchiq
check_contains \
    "$REPO_ROOT/configs/server.yaml" \
    '://patchiq:patchiq@\|:patchiq@' \
    "configs/server.yaml: database password is patchiq"

# .env.example: password in URL must be patchiq
check_contains \
    "$REPO_ROOT/.env.example" \
    '://patchiq:patchiq@\|:patchiq@' \
    ".env.example: DATABASE_URL password is patchiq"

echo ""

# --- Hub port checks ---
echo "-- Hub port checks --"

# configs/hub.yaml: port must be 8082
check_contains \
    "$REPO_ROOT/configs/hub.yaml" \
    'port:.*8082\|port: 8082' \
    "configs/hub.yaml: port is 8082"
check_not_contains \
    "$REPO_ROOT/configs/hub.yaml" \
    'port:.*8070\|port: 8070' \
    "configs/hub.yaml: port must not be 8070 (old incorrect value)"

# api/hub.yaml: servers.url must use port 8082 (was incorrectly 8090)
check_contains \
    "$REPO_ROOT/api/hub.yaml" \
    ':8082' \
    "api/hub.yaml: servers url uses port 8082"
check_not_contains \
    "$REPO_ROOT/api/hub.yaml" \
    ':8090' \
    "api/hub.yaml: servers url must not use old port 8090"

# .env.example: HUB_PORT=8082 (was incorrectly 8070)
check_contains \
    "$REPO_ROOT/.env.example" \
    'HUB_PORT=8082' \
    ".env.example: HUB_PORT=8082"
check_not_contains \
    "$REPO_ROOT/.env.example" \
    'HUB_PORT=8070' \
    ".env.example: HUB_PORT must not be old value 8070"

echo ""

# --- Init script checks ---
echo "-- Init script checks --"

check_file_exists \
    "$REPO_ROOT/deploy/docker/init-dev-dbs.sh" \
    "deploy/docker/init-dev-dbs.sh exists"

check_contains \
    "$REPO_ROOT/docker-compose.dev.yml" \
    'init-dev-dbs.sh' \
    "docker-compose.dev.yml mounts init-dev-dbs.sh"

check_contains \
    "$REPO_ROOT/deploy/docker/init-dev-dbs.sh" \
    'CREATE DATABASE patchiq_hub' \
    "init-dev-dbs.sh: creates patchiq_hub database"

echo ""

# --- Result ---
if [ "$FAILURES" -eq 0 ]; then
    echo "PASSED — all dev config checks passed."
    exit 0
else
    echo "FAILED — $FAILURES check(s) failed."
    exit 1
fi
