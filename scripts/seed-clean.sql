-- seed-clean.sql — Minimal seed for clean production start.
-- Run with: make seed-clean
-- Only creates tenant context — no demo data.

BEGIN;

INSERT INTO tenants (id, name, slug)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default')
ON CONFLICT DO NOTHING;

COMMIT;
