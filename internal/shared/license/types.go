package license

import "time"

// Customer identifies the license holder.
type Customer struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ContactEmail string `json:"contact_email"`
}

// Features describes the licensed feature set.
type Features struct {
	MaxEndpoints             int      `json:"max_endpoints"`
	OSSupport                []string `json:"os_support"`
	VisualWorkflowBuilder    bool     `json:"visual_workflow_builder"`
	AIAssistant              bool     `json:"ai_assistant"`
	SSOSAML                  bool     `json:"sso_saml"`
	SSOOIDC                  bool     `json:"sso_oidc"`
	ApprovalWorkflows        bool     `json:"approval_workflows"`
	ComplianceReports        []string `json:"compliance_reports"`
	MultiSite                bool     `json:"multi_site"`
	HADR                     bool     `json:"ha_dr"`
	CustomRBAC               bool     `json:"custom_rbac"`
	APIAccess                bool     `json:"api_access"`
	ThirdPartyPatching       bool     `json:"third_party_patching"`
	VulnerabilityIntegration bool     `json:"vulnerability_integration"`
}

// License is the full license payload (excluding signature).
type License struct {
	LicenseID           string    `json:"license_id"`
	Customer            Customer  `json:"customer"`
	Tier                string    `json:"tier"`
	Features            Features  `json:"features"`
	IssuedAt            time.Time `json:"issued_at"`
	ExpiresAt           time.Time `json:"expires_at"`
	GracePeriodDays     int       `json:"grace_period_days"`
	HardwareFingerprint *string   `json:"hardware_fingerprint"`
}

// SignedLicense is a License with its RSA signature.
type SignedLicense struct {
	License
	Signature string `json:"signature"`
}

// Tier constants.
const (
	TierCommunity    = "community"
	TierProfessional = "professional"
	TierEnterprise   = "enterprise"
	TierMSP          = "msp"
)

// LicenseStatus is the API response for GET /api/v1/license/status.
type LicenseStatus struct {
	LicenseID       string          `json:"license_id"`
	Tier            string          `json:"tier"`
	CustomerName    string          `json:"customer_name"`
	IssuedAt        time.Time       `json:"issued_at"`
	ExpiresAt       time.Time       `json:"expires_at"`
	DaysRemaining   int             `json:"days_remaining"`
	GracePeriodDays int             `json:"grace_period_days"`
	InGracePeriod   bool            `json:"in_grace_period"`
	EndpointUsage   EndpointUsage   `json:"endpoint_usage"`
	Features        map[string]bool `json:"features"`
}

// EndpointUsage tracks current vs licensed endpoint counts.
type EndpointUsage struct {
	Current int `json:"current"`
	Limit   int `json:"limit"`
}
