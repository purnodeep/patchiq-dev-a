package v1

import "github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"

// FilterEndpointsForDeploy exposes filterEndpointsForDeploy for unit tests.
var FilterEndpointsForDeploy = filterEndpointsForDeploy

// MakeEndpoint constructs a minimal sqlcgen.Endpoint for tests.
func MakeEndpoint(id, osFamily, status string) sqlcgen.Endpoint {
	ep := sqlcgen.Endpoint{OsFamily: osFamily, Status: status}
	_ = ep.ID.Scan(id)
	return ep
}
