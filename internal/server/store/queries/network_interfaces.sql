-- name: ListEndpointNetworkInterfaces :many
SELECT id, tenant_id, endpoint_id, name, ip_address, mac_address, status, created_at, updated_at
FROM endpoint_network_interfaces
WHERE tenant_id = @tenant_id AND endpoint_id = @endpoint_id
ORDER BY name
LIMIT 100;

-- name: UpsertEndpointNetworkInterface :exec
INSERT INTO endpoint_network_interfaces (tenant_id, endpoint_id, name, ip_address, mac_address, status)
VALUES (@tenant_id, @endpoint_id, @name, @ip_address, @mac_address, @status)
ON CONFLICT (tenant_id, endpoint_id, name)
    DO UPDATE SET ip_address = EXCLUDED.ip_address,
                  mac_address = EXCLUDED.mac_address,
                  status = EXCLUDED.status,
                  updated_at = now();

-- name: DeleteStaleNetworkInterfaces :exec
DELETE FROM endpoint_network_interfaces
WHERE tenant_id = @tenant_id AND endpoint_id = @endpoint_id
  AND name != ALL(@active_names::text[]);
