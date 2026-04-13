package events

// Standard event types for M0. New types are added as features are built.
// Format: resource.action
const (
	ConfigUpdated = "config.updated"

	TenantCreated = "tenant.created"
	TenantUpdated = "tenant.updated"

	CatalogCreated = "catalog.created"
	CatalogUpdated = "catalog.updated"
	CatalogDeleted = "catalog.deleted"
	CatalogSynced  = "catalog.synced"

	FeedSourceUpdated = "feed.source_updated"
	FeedSyncCompleted = "feed.sync_completed"
	FeedSyncFailed    = "feed.sync_failed"

	CVEFeedEnriched  = "cve_feed.enriched"
	CVESyncCompleted = "cve_sync.completed"
	CVESyncFailed    = "cve_sync.failed"

	PackageAliasCreated = "package_alias.created"
	PackageAliasUpdated = "package_alias.updated"
	PackageAliasDeleted = "package_alias.deleted"

	BinaryFetched     = "binary.fetched"
	BinaryFetchFailed = "binary.fetch_failed"

	ClientRegistered = "client.registered"
	ClientUpdated    = "client.updated"
	ClientApproved   = "client.approved"
	ClientDeclined   = "client.declined"
	ClientSuspended  = "client.suspended"
	ClientRemoved    = "client.removed"

	LicenseIssued   = "license.issued"
	LicenseRevoked  = "license.revoked"
	LicenseAssigned = "license.assigned"
	LicenseRenewed  = "license.renewed"

	SyncStarted   = "sync.started"
	SyncCompleted = "sync.completed"
	SyncFailed    = "sync.failed"

	AuthLogin  = "auth.login"
	AuthLogout = "auth.logout"
)

// AllTopics returns every known event topic. Used by the Watermill router
// to register subscribers, and by wildcard matching.
func AllTopics() []string {
	return []string{
		ConfigUpdated,
		TenantCreated,
		TenantUpdated,
		CatalogCreated,
		CatalogUpdated,
		CatalogDeleted,
		CatalogSynced,
		FeedSourceUpdated,
		FeedSyncCompleted,
		FeedSyncFailed,
		CVEFeedEnriched,
		CVESyncCompleted,
		CVESyncFailed,
		PackageAliasCreated,
		PackageAliasUpdated,
		PackageAliasDeleted,
		BinaryFetched,
		BinaryFetchFailed,
		ClientRegistered,
		ClientUpdated,
		ClientApproved,
		ClientDeclined,
		ClientSuspended,
		ClientRemoved,
		LicenseIssued,
		LicenseRevoked,
		LicenseAssigned,
		LicenseRenewed,
		SyncStarted,
		SyncCompleted,
		SyncFailed,
		AuthLogin,
		AuthLogout,
	}
}
