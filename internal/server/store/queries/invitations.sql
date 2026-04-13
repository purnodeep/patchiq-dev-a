-- name: CreateInvitation :one
INSERT INTO invitations (tenant_id, email, role_id, invited_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetInvitationByCode :one
-- Returns the invitation row only (no joins). This query runs on the bypass
-- pool before the tenant is known (public endpoint). After resolving the
-- tenant from the invitation, use tenant-scoped queries to look up role/tenant names.
SELECT * FROM invitations
WHERE code = $1 AND status = 'pending' AND expires_at > now();

-- name: ClaimInvitation :one
UPDATE invitations
SET status = 'claimed', claimed_at = now()
WHERE code = $1 AND status = 'pending' AND expires_at > now()
RETURNING *;

-- name: ListInvitations :many
SELECT * FROM invitations
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2;
