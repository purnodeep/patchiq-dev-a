#!/usr/bin/env bash
# deploy/docker/init-dev-dbs.sh
# Creates dev databases for all developers on the shared dev server.
# Runs automatically via /docker-entrypoint-initdb.d/ on first postgres start.
# The default POSTGRES_DB (patchiq) is created by the entrypoint itself.
set -euo pipefail

# All databases to create (beyond the default POSTGRES_DB).
# Includes legacy names for backwards compatibility + per-user databases.
DATABASES=(
    patchiq_dev
    patchiq_hub_dev
    patchiq_dev_heramb
    patchiq_hub_dev_heramb
    patchiq_dev_sandy
    patchiq_hub_dev_sandy
    patchiq_dev_danish
    patchiq_hub_dev_danish
    patchiq_dev_rishab
    patchiq_hub_dev_rishab
    patchiq_dev_production
    patchiq_hub_dev_production
)

for DB_NAME in "${DATABASES[@]}"; do
    DB_EXISTS=$(psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -Atc \
        "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" 2>&1) || {
        echo "init-dev-dbs.sh: ERROR: failed to query pg_database for $DB_NAME: $DB_EXISTS" >&2
        continue
    }

    if [ "$DB_EXISTS" = "1" ]; then
        echo "init-dev-dbs.sh: $DB_NAME already exists, skipping"
    else
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
            CREATE DATABASE "$DB_NAME";
            GRANT ALL PRIVILEGES ON DATABASE "$DB_NAME" TO "$POSTGRES_USER";
EOSQL
        echo "init-dev-dbs.sh: created $DB_NAME"
    fi
done

# Also create the patchiq_hub alias (legacy docker-compose default)
DB_EXISTS=$(psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -Atc \
    "SELECT 1 FROM pg_database WHERE datname = 'patchiq_hub'" 2>&1) || true
if [ "$DB_EXISTS" != "1" ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        CREATE DATABASE patchiq_hub;
        GRANT ALL PRIVILEGES ON DATABASE patchiq_hub TO "$POSTGRES_USER";
EOSQL
    echo "init-dev-dbs.sh: created patchiq_hub database"
fi
