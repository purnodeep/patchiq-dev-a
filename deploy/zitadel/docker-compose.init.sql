-- Creates the zitadel database and user alongside the patchiq database.
-- Mounted into postgres initdb.d so both databases exist on first boot.
CREATE USER zitadel WITH PASSWORD 'zitadel';
CREATE DATABASE zitadel OWNER zitadel;
