// TODO(#329): Wire Generator into LicenseHandler.Create to replace placeholder license keys with RSA-signed licenses.
package license

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"

	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

// GenerateParams are the inputs for license generation.
type GenerateParams struct {
	LicenseID       string
	CustomerID      string
	CustomerName    string
	ContactEmail    string
	Tier            string
	MaxEndpoints    int // 0 = use tier default
	ExpiresAt       time.Time
	GracePeriodDays int // 0 = default 30
}

// Generator signs license files with an RSA private key.
type Generator struct {
	privateKey *rsa.PrivateKey
}

// NewGenerator creates a Generator.
func NewGenerator(privateKey *rsa.PrivateKey) *Generator {
	return &Generator{privateKey: privateKey}
}

// Generate creates a signed license from the given parameters.
func (g *Generator) Generate(params GenerateParams) (*licdefs.SignedLicense, error) {
	tmpl, err := licdefs.TierTemplate(params.Tier)
	if err != nil {
		return nil, fmt.Errorf("generate license: %w", err)
	}

	if params.MaxEndpoints > 0 {
		tmpl.MaxEndpoints = params.MaxEndpoints
	}

	graceDays := params.GracePeriodDays
	if graceDays == 0 {
		graceDays = 30
	}

	lic := licdefs.License{
		LicenseID: params.LicenseID,
		Customer: licdefs.Customer{
			ID:           params.CustomerID,
			Name:         params.CustomerName,
			ContactEmail: params.ContactEmail,
		},
		Tier:            params.Tier,
		Features:        tmpl,
		IssuedAt:        time.Now().UTC().Truncate(time.Second),
		ExpiresAt:       params.ExpiresAt.UTC().Truncate(time.Second),
		GracePeriodDays: graceDays,
	}

	canonical, err := json.Marshal(lic)
	if err != nil {
		return nil, fmt.Errorf("generate license: marshal canonical: %w", err)
	}

	sig, err := crypto.SignPayload(g.privateKey, canonical)
	if err != nil {
		return nil, fmt.Errorf("generate license: %w", err)
	}

	return &licdefs.SignedLicense{
		License:   lic,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}, nil
}

// SaveToFile writes a signed license to disk as JSON.
func SaveToFile(lic *licdefs.SignedLicense, path string) error {
	data, err := json.MarshalIndent(lic, "", "  ")
	if err != nil {
		return fmt.Errorf("save license file: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("save license file: %w", err)
	}
	return nil
}

// DecodeSignature decodes a base64-encoded signature.
func DecodeSignature(sig string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	return data, nil
}
