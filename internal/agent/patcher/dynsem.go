package patcher

import "sync"

// dynamicSem is a semaphore whose maximum concurrency can change at runtime
// via the maxFunc callback. It uses a condition variable instead of a channel
// so the limit can be adjusted without recreating the semaphore.
type dynamicSem struct {
	mu      sync.Mutex
	cond    *sync.Cond
	current int
	maxFunc func() int
}

// newDynamicSem creates a dynamicSem. maxFunc is called on every Acquire to
// read the current concurrency limit. It must be safe to call from multiple
// goroutines.
func newDynamicSem(maxFunc func() int) *dynamicSem {
	s := &dynamicSem{maxFunc: maxFunc}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Acquire blocks until the number of concurrent holders is below the current
// max returned by maxFunc. The limit is re-evaluated on each iteration so
// runtime changes take effect immediately.
func (s *dynamicSem) Acquire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		max := s.maxFunc()
		if max < 1 {
			max = 1
		}
		if s.current < max {
			break
		}
		s.cond.Wait()
	}
	s.current++
}

// Release decrements the holder count and wakes all waiters so they can
// re-check the (possibly changed) limit.
func (s *dynamicSem) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current--
	s.cond.Broadcast()
}
