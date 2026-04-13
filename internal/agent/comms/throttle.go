package comms

import (
	"sync"
	"time"
)

// Throttler rate-limits outbox message sending based on a bandwidth limit in
// kilobits per second. It uses a token bucket algorithm where tokens represent
// bytes. If the limit is 0, no throttling is applied.
type Throttler struct {
	mu       sync.Mutex
	limit    int64 // bytes per second, 0 = unlimited
	tokens   int64
	lastFill time.Time
}

// NewThrottler creates a Throttler with the given bandwidth limit in Kbps.
// A limit of 0 means unlimited (no throttling).
func NewThrottler(kbps int) *Throttler {
	bps := kbpsToBytesPerSec(kbps)
	return &Throttler{
		limit:    bps,
		tokens:   bps, // start with a full bucket (1 second worth)
		lastFill: time.Now(),
	}
}

// SetLimit updates the bandwidth limit at runtime. Thread-safe.
func (t *Throttler) SetLimit(kbps int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.limit = kbpsToBytesPerSec(kbps)
	// Reset bucket so new limit takes effect immediately.
	t.tokens = t.limit
	t.lastFill = time.Now()
}

// Throttle blocks for the appropriate duration before a message of the given
// size (in bytes) should be sent. Returns immediately if the limit is 0.
func (t *Throttler) Throttle(messageSize int) {
	if messageSize <= 0 {
		return
	}

	t.mu.Lock()

	if t.limit <= 0 {
		t.mu.Unlock()
		return
	}

	// Refill tokens based on time elapsed since last fill.
	now := time.Now()
	elapsed := now.Sub(t.lastFill)
	refill := int64(elapsed.Seconds() * float64(t.limit))
	t.tokens += refill
	// Cap tokens at 1 second worth (max burst).
	if t.tokens > t.limit {
		t.tokens = t.limit
	}
	t.lastFill = now

	needed := int64(messageSize)
	if t.tokens >= needed {
		t.tokens -= needed
		t.mu.Unlock()
		return
	}

	// Not enough tokens — calculate how long to wait.
	deficit := needed - t.tokens
	t.tokens = 0
	waitDuration := time.Duration(float64(deficit) / float64(t.limit) * float64(time.Second))
	t.lastFill = now.Add(waitDuration)

	// Release lock while sleeping to avoid holding it for the wait duration.
	t.mu.Unlock()
	time.Sleep(waitDuration)
}

// kbpsToBytesPerSec converts kilobits per second to bytes per second.
// 1 Kbps = 1000 bits/s = 125 bytes/s.
func kbpsToBytesPerSec(kbps int) int64 {
	if kbps <= 0 {
		return 0
	}
	return int64(kbps) * 125
}
