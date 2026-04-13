package comms

import (
	"sync"
	"testing"
	"time"
)

func TestThrottle_Unlimited(t *testing.T) {
	t.Parallel()

	th := NewThrottler(0) // unlimited

	start := time.Now()
	for i := 0; i < 10; i++ {
		th.Throttle(100_000) // 100 KB each call — would take seconds if throttled
	}
	elapsed := time.Since(start)

	// With unlimited mode all calls should return nearly instantly.
	if elapsed > 50*time.Millisecond {
		t.Errorf("unlimited mode took %v, want < 50ms", elapsed)
	}
}

func TestThrottle_ZeroMessageSize(t *testing.T) {
	t.Parallel()

	// Even with a very tight limit, a zero-size message must return immediately.
	th := NewThrottler(1) // 1 Kbps = 125 bytes/sec

	start := time.Now()
	th.Throttle(0)
	th.Throttle(-1)
	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Errorf("zero/negative message size took %v, want < 10ms", elapsed)
	}
}

func TestThrottle_SmallMessageConsumesTokens(t *testing.T) {
	t.Parallel()

	// 8 Kbps = 1000 bytes/sec. Bucket starts full (1000 tokens).
	th := NewThrottler(8)

	// First small message fits in the initial bucket — should not block.
	start := time.Now()
	th.Throttle(100) // 100 bytes < 1000 token bucket
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Errorf("small message (tokens available) took %v, want < 20ms", elapsed)
	}
}

func TestThrottle_RefillOverTime(t *testing.T) {
	t.Parallel()

	// 8 Kbps = 1000 bytes/sec.
	// Drain the bucket completely, then verify a subsequent call blocks
	// for roughly the refill time.
	th := NewThrottler(8)

	// Drain entire 1-second bucket immediately.
	th.Throttle(1000) // uses all tokens, no wait (bucket was full)

	// Next 500-byte message requires ~500ms refill.
	start := time.Now()
	th.Throttle(500)
	elapsed := time.Since(start)

	// Allow generous tolerance (±200ms) for CI timing jitter.
	if elapsed < 200*time.Millisecond {
		t.Errorf("expected throttle wait ≥200ms, got %v", elapsed)
	}
	if elapsed > 900*time.Millisecond {
		t.Errorf("throttle wait too long: %v (want < 900ms)", elapsed)
	}
}

func TestThrottle_SetLimit_UpdatesMidUse(t *testing.T) {
	t.Parallel()

	// Start with a tight limit.
	th := NewThrottler(8) // 1000 bytes/sec

	// Switch to unlimited — subsequent calls should not block.
	th.SetLimit(0)

	start := time.Now()
	th.Throttle(100_000) // would take 100s at the old rate
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Errorf("after SetLimit(0) call took %v, want < 20ms", elapsed)
	}

	// Switch to a moderate limit and verify the bucket resets.
	th.SetLimit(8) // 1000 bytes/sec again, fresh bucket
	start = time.Now()
	th.Throttle(500) // 500 bytes — fits in the newly reset 1000-token bucket
	if elapsed := time.Since(start); elapsed > 30*time.Millisecond {
		t.Errorf("after SetLimit fresh-bucket call took %v, want < 30ms", elapsed)
	}
}

func TestThrottle_Concurrent(t *testing.T) {
	t.Parallel()

	// Use unlimited so goroutines don't actually sleep — we just want to
	// exercise the mutex under the race detector.
	th := NewThrottler(0)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			th.Throttle(1024)
		}()
	}
	wg.Wait()
}

func TestThrottle_Concurrent_WithLimit(t *testing.T) {
	t.Parallel()

	// Use a small but nonzero limit to exercise the throttled path concurrently.
	// 80 Kbps = 10000 bytes/sec — large enough that small messages don't sleep long.
	th := NewThrottler(80)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Each goroutine sends a 10-byte message — well within refill.
			th.Throttle(10)
		}()
	}
	// Just verify no race and no panic — the goroutines may or may not block
	// depending on timing, but all should complete quickly.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent throttle calls did not complete within 2s")
	}
}
