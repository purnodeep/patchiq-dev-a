-- name: CreateRegistration :one
INSERT INTO agent_registrations (tenant_id, registration_token, expires_at)
VALUES (@tenant_id, @registration_token, COALESCE(sqlc.narg('expires_at')::timestamptz, now() + INTERVAL '7 days'))
RETURNING *;

-- name: GetRegistrationByID :one
SELECT * FROM agent_registrations WHERE id = $1 AND tenant_id = $2;

-- name: GetRegistrationByToken :one
SELECT * FROM agent_registrations WHERE registration_token = $1 AND tenant_id = $2;

-- name: ListRegistrationsByTenant :many
SELECT * FROM agent_registrations WHERE tenant_id = $1 ORDER BY created_at DESC
LIMIT 1000;

-- name: ClaimRegistration :one
UPDATE agent_registrations
SET status = 'registered', endpoint_id = $3, registered_at = now()
WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
RETURNING *;

-- name: LookupRegistrationByToken :one
SELECT * FROM agent_registrations WHERE registration_token = $1;

-- name: RevokeRegistration :one
UPDATE agent_registrations
SET status = 'revoked'
WHERE id = $1 AND tenant_id = $2 AND status != 'revoked'
RETURNING *;
