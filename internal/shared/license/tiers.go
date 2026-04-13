package license

import "fmt"

// TierTemplate returns the default Features for a given tier.
func TierTemplate(tier string) (Features, error) {
	switch tier {
	case TierCommunity:
		return Features{
			MaxEndpoints:             25,
			OSSupport:                []string{"linux"},
			VisualWorkflowBuilder:    true,
			AIAssistant:              false,
			SSOSAML:                  false,
			SSOOIDC:                  false,
			ApprovalWorkflows:        false,
			ComplianceReports:        []string{},
			MultiSite:                false,
			HADR:                     false,
			CustomRBAC:               false,
			APIAccess:                false,
			ThirdPartyPatching:       false,
			VulnerabilityIntegration: false,
		}, nil
	case TierProfessional:
		return Features{
			MaxEndpoints:             1000,
			OSSupport:                []string{"windows", "linux", "macos"},
			VisualWorkflowBuilder:    true,
			AIAssistant:              true,
			SSOSAML:                  false,
			SSOOIDC:                  false,
			ApprovalWorkflows:        true,
			ComplianceReports:        []string{"soc2"},
			MultiSite:                false,
			HADR:                     false,
			CustomRBAC:               false,
			APIAccess:                true,
			ThirdPartyPatching:       true,
			VulnerabilityIntegration: true,
		}, nil
	case TierEnterprise:
		return Features{
			MaxEndpoints:             10000,
			OSSupport:                []string{"windows", "linux", "macos"},
			VisualWorkflowBuilder:    true,
			AIAssistant:              true,
			SSOSAML:                  true,
			SSOOIDC:                  true,
			ApprovalWorkflows:        true,
			ComplianceReports:        []string{"hipaa", "soc2", "iso27001"},
			MultiSite:                true,
			HADR:                     true,
			CustomRBAC:               true,
			APIAccess:                true,
			ThirdPartyPatching:       true,
			VulnerabilityIntegration: true,
		}, nil
	case TierMSP:
		return Features{
			MaxEndpoints:             0, // unlimited
			OSSupport:                []string{"windows", "linux", "macos"},
			VisualWorkflowBuilder:    true,
			AIAssistant:              true,
			SSOSAML:                  true,
			SSOOIDC:                  true,
			ApprovalWorkflows:        true,
			ComplianceReports:        []string{"hipaa", "soc2", "iso27001"},
			MultiSite:                true,
			HADR:                     true,
			CustomRBAC:               true,
			APIAccess:                true,
			ThirdPartyPatching:       true,
			VulnerabilityIntegration: true,
		}, nil
	default:
		return Features{}, fmt.Errorf("unknown license tier: %q", tier)
	}
}

// FeatureMap converts a Features struct to a map[string]bool for easy lookup.
func FeatureMap(f Features) map[string]bool {
	return map[string]bool{
		"visual_workflow_builder":   f.VisualWorkflowBuilder,
		"ai_assistant":              f.AIAssistant,
		"sso_saml":                  f.SSOSAML,
		"sso_oidc":                  f.SSOOIDC,
		"approval_workflows":        f.ApprovalWorkflows,
		"multi_site":                f.MultiSite,
		"ha_dr":                     f.HADR,
		"custom_rbac":               f.CustomRBAC,
		"api_access":                f.APIAccess,
		"third_party_patching":      f.ThirdPartyPatching,
		"vulnerability_integration": f.VulnerabilityIntegration,
	}
}
