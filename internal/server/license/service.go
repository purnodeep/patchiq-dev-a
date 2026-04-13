package license

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

// TODO(PIQ-331): add gRPC client to call hub's ValidateLicense RPC (H-I2) — server
// currently validates licenses locally but should also validate against the hub
// for online deployments to prevent license reuse across Patch Manager instances.

// Service manages the runtime license state for Patch Manager.
type Service struct {
	validator *Validator
	eventBus  domain.EventBus
	now       func() time.Time

	mu      sync.RWMutex
	license *licdefs.License
}

// NewService creates a license Service.
func NewService(validator *Validator, eventBus domain.EventBus) *Service {
	return &Service{
		validator: validator,
		eventBus:  eventBus,
		now:       time.Now,
	}
}

// WithClock sets a custom clock function (for testing).
func (s *Service) WithClock(now func() time.Time) {
	s.now = now
	s.validator.WithClock(now)
}

// LoadFromFile reads, validates, and caches a license file.
func (s *Service) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load license file: %w", err)
	}

	lic, err := s.validator.Validate(data)
	if err != nil {
		// ErrInGracePeriod is not fatal — license is still usable
		if lic == nil {
			return fmt.Errorf("load license file: %w", err)
		}
		slog.Warn("license in grace period", "license_id", lic.LicenseID, "error", err)
		s.emitEvent("license.grace_period_entered", lic.LicenseID)
	}

	s.mu.Lock()
	s.license = lic
	s.mu.Unlock()

	s.emitEvent("license.loaded", lic.LicenseID)

	// Check if expiring soon (< 30 days)
	daysRemaining := int(time.Until(lic.ExpiresAt).Hours() / 24)
	if daysRemaining > 0 && daysRemaining <= 30 {
		s.emitEvent("license.expiring", lic.LicenseID)
	}

	return nil
}

// HasFeature checks if the current license includes a named feature.
// Returns false if no license is loaded (community default).
func (s *Service) HasFeature(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.license == nil {
		return false
	}

	fm := licdefs.FeatureMap(s.license.Features)
	return fm[name]
}

// CheckEndpointLimit returns an error if current exceeds the licensed limit.
// A limit of 0 means unlimited (MSP tier).
func (s *Service) CheckEndpointLimit(current int) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := 25 // community default
	if s.license != nil {
		limit = s.license.Features.MaxEndpoints
	}

	if limit == 0 {
		return nil // unlimited
	}

	if current > limit {
		return fmt.Errorf("endpoint limit exceeded: %d/%d", current, limit)
	}
	return nil
}

// CurrentTier returns the current license tier name.
func (s *Service) CurrentTier() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.license == nil {
		return licdefs.TierCommunity
	}
	return s.license.Tier
}

// IsExpired returns true if the license is past its expiry (excluding grace).
func (s *Service) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.license == nil {
		return false
	}
	return s.now().After(s.license.ExpiresAt.Add(clockDriftTolerance))
}

// InGracePeriod returns true if expired but within the grace window.
func (s *Service) InGracePeriod() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inGracePeriodUnlocked()
}

func (s *Service) inGracePeriodUnlocked() bool {
	if s.license == nil {
		return false
	}
	now := s.now()
	pastExpiry := now.After(s.license.ExpiresAt.Add(clockDriftTolerance))
	graceEnd := s.license.ExpiresAt.Add(time.Duration(s.license.GracePeriodDays) * 24 * time.Hour)
	return pastExpiry && now.Before(graceEnd)
}

// Status returns the full license status snapshot.
func (s *Service) Status() licdefs.LicenseStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.license == nil {
		communityFeatures, _ := licdefs.TierTemplate(licdefs.TierCommunity)
		return licdefs.LicenseStatus{
			Tier:          licdefs.TierCommunity,
			DaysRemaining: 0,
			EndpointUsage: licdefs.EndpointUsage{Limit: 25},
			Features:      licdefs.FeatureMap(communityFeatures),
		}
	}

	daysRemaining := max(int(math.Ceil(time.Until(s.license.ExpiresAt).Hours()/24)), 0)

	return licdefs.LicenseStatus{
		LicenseID:       s.license.LicenseID,
		Tier:            s.license.Tier,
		CustomerName:    s.license.Customer.Name,
		IssuedAt:        s.license.IssuedAt,
		ExpiresAt:       s.license.ExpiresAt,
		DaysRemaining:   daysRemaining,
		GracePeriodDays: s.license.GracePeriodDays,
		InGracePeriod:   s.inGracePeriodUnlocked(),
		EndpointUsage:   licdefs.EndpointUsage{Limit: s.license.Features.MaxEndpoints},
		Features:        licdefs.FeatureMap(s.license.Features),
	}
}

func (s *Service) emitEvent(eventType, licenseID string) {
	if s.eventBus == nil {
		return
	}
	event := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       eventType,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "license",
		ResourceID: licenseID,
		Action:     eventType,
		Timestamp:  s.now(),
	}
	if err := s.eventBus.Emit(context.Background(), event); err != nil {
		slog.Error("emit license event failed", "event_type", eventType, "license_id", licenseID, "error", err)
	}
}
