package auth

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrSessionNotFound indicates no session exists for the user.
var ErrSessionNotFound = errors.New("session not found")

// RefreshTokenStore manages refresh token persistence.
type RefreshTokenStore interface {
	StoreRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
}

type memEntry struct {
	token     string
	expiresAt time.Time
}

// MemorySessionStore is an in-memory RefreshTokenStore for dev/testing.
type MemorySessionStore struct {
	mu    sync.RWMutex
	store map[string]memEntry
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{store: make(map[string]memEntry)}
}

func (s *MemorySessionStore) StoreRefreshToken(_ context.Context, userID, token string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[userID] = memEntry{token: token, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (s *MemorySessionStore) GetRefreshToken(_ context.Context, userID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.store[userID]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", ErrSessionNotFound
	}
	return entry.token, nil
}

func (s *MemorySessionStore) DeleteRefreshToken(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, userID)
	return nil
}
