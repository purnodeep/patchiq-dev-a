package v1

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLimit int32 = 50
	minLimit     int32 = 1
	maxLimit     int32 = 200
)

// EncodeCursor encodes a (created_at, id) pair into an opaque base64 cursor string.
func EncodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%d|%s", createdAt.UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor string back into (created_at, id).
// Returns zero values and nil error for an empty cursor (first page).
func DecodeCursor(cursor string) (time.Time, string, error) {
	if cursor == "" {
		return time.Time{}, "", nil
	}
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("decode cursor: invalid base64: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("decode cursor: expected format 'timestamp|uuid', got %q", string(raw))
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("decode cursor: invalid timestamp: %w", err)
	}
	return time.Unix(0, nanos), parts[1], nil
}

// ParseLimit parses a limit query parameter, clamping to [1, 200] with default 50.
func ParseLimit(raw string) int32 {
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return defaultLimit
	}
	if int32(n) < minLimit {
		return minLimit
	}
	if int32(n) > maxLimit {
		return maxLimit
	}
	return int32(n)
}
