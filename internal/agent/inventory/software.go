package inventory

// ExtendedPackageInfo holds enriched metadata beyond what the gRPC PackageInfo
// proto message supports. This data is stored in-memory and served via the
// local agent HTTP API for detailed software inventory views.
type ExtendedPackageInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Architecture  string `json:"architecture"`
	Source        string `json:"source"`
	Status        string `json:"status"`
	InstalledSize int    `json:"installed_size_kb,omitempty"`
	Maintainer    string `json:"maintainer,omitempty"`
	Section       string `json:"section,omitempty"`
	Homepage      string `json:"homepage,omitempty"`
	Description   string `json:"description,omitempty"`
	InstallDate   string `json:"install_date,omitempty"`
	License       string `json:"license,omitempty"`
	Priority      string `json:"priority,omitempty"`
	SourcePackage string `json:"source_package,omitempty"`
	Category      string `json:"category,omitempty"`
}

// ServiceInfo represents a systemd service unit collected from the endpoint.
type ServiceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	Enabled     bool   `json:"enabled"`
	Category    string `json:"category"`
}
