# Zitadel Machine Keys

This directory is mounted as a volume in docker-compose.
Zitadel writes machine key JSON files here on first boot.
The server uses these keys to authenticate as a service account
for the Zitadel Management API (user sync job).

Files in this directory are gitignored (generated at runtime).
