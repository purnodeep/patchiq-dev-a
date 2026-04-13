-- name: UpsertUserIdentity :one
INSERT INTO user_identities (tenant_id, external_id, provider, email, display_name, last_login_at)
VALUES (@tenant_id, @external_id, @provider, @email, @display_name, now())
ON CONFLICT (tenant_id, external_id, provider)
DO UPDATE SET
    email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    last_login_at = now()
RETURNING *;

-- name: GetUserIdentityByExternalID :one
SELECT * FROM user_identities
WHERE tenant_id = @tenant_id AND external_id = @external_id AND provider = @provider;

-- name: ListUserIdentities :many
SELECT * FROM user_identities
WHERE tenant_id = @tenant_id AND disabled = false
ORDER BY display_name
LIMIT 1000;

-- name: DisableUserIdentity :exec
UPDATE user_identities SET disabled = true
WHERE tenant_id = @tenant_id AND external_id = @external_id AND provider = @provider;

-- name: ListRoleMappings :many
SELECT rm.*, r.name as role_name
FROM role_mappings rm
JOIN roles r ON r.id = rm.patchiq_role_id
WHERE rm.tenant_id = @tenant_id
ORDER BY rm.external_role
LIMIT 1000;

-- name: UpsertRoleMapping :one
INSERT INTO role_mappings (tenant_id, external_role, patchiq_role_id)
VALUES (@tenant_id, @external_role, @patchiq_role_id)
ON CONFLICT (tenant_id, external_role)
DO UPDATE SET patchiq_role_id = EXCLUDED.patchiq_role_id, updated_at = now()
RETURNING *;

-- name: DeleteRoleMapping :exec
DELETE FROM role_mappings
WHERE tenant_id = @tenant_id AND id = @id;

-- name: DeleteRoleMappingsByTenant :exec
DELETE FROM role_mappings
WHERE tenant_id = @tenant_id;

-- name: GetRoleMappingByExternalRole :one
SELECT * FROM role_mappings
WHERE tenant_id = @tenant_id AND external_role = @external_role;

-- name: GetIAMSettings :one
SELECT tenant_id, zitadel_org_id, default_role_id, user_sync_enabled, user_sync_interval,
       sso_url, client_id_encrypted, last_test_status, last_tested_at,
       created_at, updated_at
FROM iam_settings
WHERE tenant_id = @tenant_id;

-- name: UpsertIAMSettings :one
INSERT INTO iam_settings (tenant_id, zitadel_org_id, default_role_id, user_sync_enabled, user_sync_interval, sso_url, client_id_encrypted)
VALUES (@tenant_id, @zitadel_org_id, @default_role_id, @user_sync_enabled, @user_sync_interval, @sso_url, @client_id_encrypted)
ON CONFLICT (tenant_id)
DO UPDATE SET
    zitadel_org_id = EXCLUDED.zitadel_org_id,
    default_role_id = EXCLUDED.default_role_id,
    user_sync_enabled = EXCLUDED.user_sync_enabled,
    user_sync_interval = EXCLUDED.user_sync_interval,
    sso_url = EXCLUDED.sso_url,
    client_id_encrypted = EXCLUDED.client_id_encrypted,
    updated_at = now()
RETURNING *;

-- name: UpdateIAMTestResult :exec
UPDATE iam_settings
SET last_test_status = @last_test_status,
    last_tested_at = @last_tested_at,
    updated_at = now()
WHERE tenant_id = @tenant_id;
