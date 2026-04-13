package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"
)

// CachedResponse holds the serialized HTTP response for an idempotency key.
type CachedResponse struct {
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type"`
	Body        []byte `json:"body"`
}

// Store is the idempotency cache interface.
type Store interface {
	Get(ctx context.Context, tenantID, key string) (CachedResponse, bool, error)
	Set(ctx context.Context, tenantID, key string, resp CachedResponse, ttl time.Duration) error
}

// cacheKey builds the namespaced cache key.
func cacheKey(tenantID, key string) string {
	return fmt.Sprintf("idempotency:%s:%s", tenantID, key)
}

// ---------------------------------------------------------------------------
// ValkeyStore
// ---------------------------------------------------------------------------

// ValkeyStore is a Store backed by Valkey (Redis-compatible).
type ValkeyStore struct {
	client valkey.Client
}

// NewValkeyStore creates a ValkeyStore with the given Valkey client.
func NewValkeyStore(client valkey.Client) *ValkeyStore {
	return &ValkeyStore{client: client}
}

// Get retrieves a cached response from Valkey. Returns found=false on a cache miss.
func (s *ValkeyStore) Get(ctx context.Context, tenantID, key string) (CachedResponse, bool, error) {
	k := cacheKey(tenantID, key)
	cmd := s.client.B().Get().Key(k).Build()
	result := s.client.Do(ctx, cmd)

	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return CachedResponse{}, false, nil
		}
		return CachedResponse{}, false, fmt.Errorf("idempotency cache get %q: %w", k, err)
	}

	raw, err := result.AsBytes()
	if err != nil {
		return CachedResponse{}, false, fmt.Errorf("idempotency cache get %q read bytes: %w", k, err)
	}

	var resp CachedResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return CachedResponse{}, false, fmt.Errorf("idempotency cache get %q unmarshal: %w", k, err)
	}

	slog.DebugContext(ctx, "idempotency cache hit", "key", k)
	return resp, true, nil
}

// Set stores a response in Valkey with the given TTL.
func (s *ValkeyStore) Set(ctx context.Context, tenantID, key string, resp CachedResponse, ttl time.Duration) error {
	k := cacheKey(tenantID, key)

	raw, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("idempotency cache set %q marshal: %w", k, err)
	}

	seconds := int64(ttl.Seconds())
	cmd := s.client.B().Set().Key(k).Value(string(raw)).Ex(time.Duration(seconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("idempotency cache set %q: %w", k, err)
	}

	slog.DebugContext(ctx, "idempotency cache stored", "key", k, "ttl_seconds", seconds)
	return nil
}

// ---------------------------------------------------------------------------
// MemoryStore (testing only)
// ---------------------------------------------------------------------------

type memoryEntry struct {
	resp CachedResponse
}

// MemoryStore is an in-memory Store used for unit tests.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]memoryEntry
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]memoryEntry)}
}

// Get retrieves a cached response from the in-memory store.
func (m *MemoryStore) Get(_ context.Context, tenantID, key string) (CachedResponse, bool, error) {
	k := cacheKey(tenantID, key)
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.data[k]
	if !ok {
		return CachedResponse{}, false, nil
	}
	return entry.resp, true, nil
}

// Set stores a response in the in-memory store. TTL is accepted but not enforced.
func (m *MemoryStore) Set(_ context.Context, tenantID, key string, resp CachedResponse, _ time.Duration) error {
	k := cacheKey(tenantID, key)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[k] = memoryEntry{resp: resp}
	return nil
}
