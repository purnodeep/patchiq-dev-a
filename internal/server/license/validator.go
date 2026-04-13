package license

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

var (
	ErrInvalidSignature = errors.New("invalid license signature")
	ErrExpired          = errors.New("license expired")
	ErrInGracePeriod    = errors.New("license in grace period")
)

const clockDriftTolerance = 48 * time.Hour

// Validator verifies license file signatures and expiry.
type Validator struct {
	publicKey *rsa.PublicKey
	now       func() time.Time // injectable clock for testing
}

// NewValidator creates a Validator with the given RSA public key.
func NewValidator(publicKey *rsa.PublicKey) *Validator {
	return &Validator{publicKey: publicKey, now: time.Now}
}

// WithClock sets a custom clock function (for testing).
func (v *Validator) WithClock(now func() time.Time) {
	v.now = now
}

// Validate parses and verifies a signed license file.
// Returns the License if the signature is valid.
// Returns ErrInGracePeriod (wrapped) if in the grace window — the license is still usable but caller should warn.
// Returns ErrExpired if past the grace period.
// Returns ErrInvalidSignature if the signature doesn't match.
func (v *Validator) Validate(data []byte) (*licdefs.License, error) {
	var signed licdefs.SignedLicense
	if err := json.Unmarshal(data, &signed); err != nil {
		return nil, fmt.Errorf("validate license: parse JSON: %w", err)
	}

	// Decode base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil {
		return nil, fmt.Errorf("validate license: decode signature: %w", err)
	}

	// Reconstruct canonical payload (License without Signature field)
	canonical, err := json.Marshal(signed.License)
	if err != nil {
		return nil, fmt.Errorf("validate license: marshal canonical: %w", err)
	}

	// Verify RSA signature
	if err := crypto.VerifySignature(v.publicKey, canonical, sigBytes); err != nil {
		return nil, fmt.Errorf("validate license: %w: %w", ErrInvalidSignature, err)
	}

	// Check expiry
	now := v.now()
	graceEnd := signed.ExpiresAt.Add(time.Duration(signed.GracePeriodDays) * 24 * time.Hour)

	if now.After(graceEnd) {
		return nil, fmt.Errorf("validate license %s: %w (expired %s, grace ended %s)",
			signed.LicenseID, ErrExpired, signed.ExpiresAt.Format(time.RFC3339), graceEnd.Format(time.RFC3339))
	}

	if now.After(signed.ExpiresAt.Add(clockDriftTolerance)) {
		// In grace period — return the license but wrap ErrInGracePeriod
		return &signed.License, fmt.Errorf("validate license %s: %w", signed.LicenseID, ErrInGracePeriod)
	}

	return &signed.License, nil
}
