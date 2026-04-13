package license

import (
	"testing"
)

func TestTierTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tier        string
		wantMax     int
		wantSSOSAML bool
		wantSSOOIDC bool
	}{
		{
			name:        "community has 25 endpoints and no SSO",
			tier:        TierCommunity,
			wantMax:     25,
			wantSSOSAML: false,
			wantSSOOIDC: false,
		},
		{
			name:        "professional has 1000 endpoints and no SSO",
			tier:        TierProfessional,
			wantMax:     1000,
			wantSSOSAML: false,
			wantSSOOIDC: false,
		},
		{
			name:        "enterprise has 10000 endpoints and full SSO",
			tier:        TierEnterprise,
			wantMax:     10000,
			wantSSOSAML: true,
			wantSSOOIDC: true,
		},
		{
			name:        "msp has unlimited endpoints and full SSO",
			tier:        TierMSP,
			wantMax:     0,
			wantSSOSAML: true,
			wantSSOOIDC: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := TierTemplate(tt.tier)
			if err != nil {
				t.Fatalf("TierTemplate(%q) returned unexpected error: %v", tt.tier, err)
			}
			if got.MaxEndpoints != tt.wantMax {
				t.Errorf("MaxEndpoints = %d, want %d", got.MaxEndpoints, tt.wantMax)
			}
			if got.SSOSAML != tt.wantSSOSAML {
				t.Errorf("SSOSAML = %v, want %v", got.SSOSAML, tt.wantSSOSAML)
			}
			if got.SSOOIDC != tt.wantSSOOIDC {
				t.Errorf("SSOOIDC = %v, want %v", got.SSOOIDC, tt.wantSSOOIDC)
			}
		})
	}
}

func TestTierTemplateInvalid(t *testing.T) {
	t.Parallel()

	_, err := TierTemplate("invalid-tier")
	if err == nil {
		t.Fatal("TierTemplate(\"invalid-tier\") expected error, got nil")
	}

	want := `unknown license tier: "invalid-tier"`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestHasFeature(t *testing.T) {
	t.Parallel()

	features, err := TierTemplate(TierEnterprise)
	if err != nil {
		t.Fatalf("TierTemplate(%q) returned unexpected error: %v", TierEnterprise, err)
	}

	fm := FeatureMap(features)

	tests := []struct {
		name    string
		feature string
		want    bool
	}{
		{"enterprise has sso_saml", "sso_saml", true},
		{"enterprise has api_access", "api_access", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := fm[tt.feature]
			if !ok {
				t.Fatalf("feature %q not found in FeatureMap", tt.feature)
			}
			if got != tt.want {
				t.Errorf("FeatureMap[%q] = %v, want %v", tt.feature, got, tt.want)
			}
		})
	}
}

func TestCommunityFeatureMap(t *testing.T) {
	t.Parallel()

	features, err := TierTemplate(TierCommunity)
	if err != nil {
		t.Fatalf("TierTemplate(%q) returned unexpected error: %v", TierCommunity, err)
	}

	fm := FeatureMap(features)

	tests := []struct {
		name    string
		feature string
		want    bool
	}{
		{"community lacks sso_saml", "sso_saml", false},
		{"community lacks multi_site", "multi_site", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := fm[tt.feature]
			if !ok {
				t.Fatalf("feature %q not found in FeatureMap", tt.feature)
			}
			if got != tt.want {
				t.Errorf("FeatureMap[%q] = %v, want %v", tt.feature, got, tt.want)
			}
		})
	}
}
