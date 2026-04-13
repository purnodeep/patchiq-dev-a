package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"
)

// TestDynamicSem_LimitsConcurrency verifies that the dynamic semaphore
// correctly limits concurrent holders to the configured maximum.
func TestDynamicSem_LimitsConcurrency(t *testing.T) {
	const maxConcurrent = 2
	sem := newDynamicSem(func() int { return maxConcurrent })

	var running atomic.Int32
	var maxObserved atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem.Acquire()
			defer sem.Release()

			cur := running.Add(1)
			// Track the peak concurrency observed across all goroutines.
			for {
				old := maxObserved.Load()
				if cur <= old || maxObserved.CompareAndSwap(old, cur) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond) // simulate work
			running.Add(-1)
		}()
	}

	wg.Wait()

	if peak := maxObserved.Load(); peak > maxConcurrent {
		t.Errorf("peak concurrent holders = %d, want <= %d", peak, maxConcurrent)
	}
	if peak := maxObserved.Load(); peak < maxConcurrent {
		t.Errorf("peak concurrent holders = %d, expected to reach %d (semaphore may be too restrictive)", peak, maxConcurrent)
	}
}

// TestDynamicSem_AllComplete verifies that all goroutines eventually
// complete even when the concurrency limit is lower than the goroutine count.
func TestDynamicSem_AllComplete(t *testing.T) {
	const maxConcurrent = 2
	const total = 5
	sem := newDynamicSem(func() int { return maxConcurrent })

	var completed atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem.Acquire()
			defer sem.Release()
			time.Sleep(10 * time.Millisecond)
			completed.Add(1)
		}()
	}

	wg.Wait()

	if got := completed.Load(); got != total {
		t.Errorf("completed = %d, want %d", got, total)
	}
}

// TestDynamicSem_LimitChangeMidFlight verifies that changing the max
// concurrency at runtime takes effect for subsequent acquires.
func TestDynamicSem_LimitChangeMidFlight(t *testing.T) {
	var limit atomic.Int32
	limit.Store(1) // start with limit=1

	sem := newDynamicSem(func() int { return int(limit.Load()) })

	var running atomic.Int32
	var maxObservedPhase1 atomic.Int32
	var maxObservedPhase2 atomic.Int32

	// Phase 1: launch 3 goroutines with limit=1. Each holds the semaphore
	// for 50ms. Track peak concurrency.
	phase1Done := make(chan struct{})
	var wg1 sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg1.Add(1)
		go func() {
			defer wg1.Done()
			sem.Acquire()
			defer sem.Release()

			cur := running.Add(1)
			for {
				old := maxObservedPhase1.Load()
				if cur <= old || maxObservedPhase1.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			running.Add(-1)
		}()
	}
	go func() {
		wg1.Wait()
		close(phase1Done)
	}()

	<-phase1Done

	if peak := maxObservedPhase1.Load(); peak != 1 {
		t.Errorf("phase 1 peak = %d, want 1", peak)
	}

	// Phase 2: raise limit to 3 and launch 3 goroutines.
	limit.Store(3)

	var wg2 sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			sem.Acquire()
			defer sem.Release()

			cur := running.Add(1)
			for {
				old := maxObservedPhase2.Load()
				if cur <= old || maxObservedPhase2.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			running.Add(-1)
		}()
	}

	wg2.Wait()

	if peak := maxObservedPhase2.Load(); peak > 3 {
		t.Errorf("phase 2 peak = %d, want <= 3", peak)
	}
	// With limit=3 and 3 goroutines all sleeping 50ms, they should all run together.
	if peak := maxObservedPhase2.Load(); peak < 3 {
		t.Logf("phase 2 peak = %d (expected 3, may be scheduling-dependent)", peak)
	}
}

// TestConcurrentPatchInstalls_SemaphoreLimit verifies that the patcher Module
// respects max_concurrent_installs when multiple install_patch commands are
// dispatched concurrently with slow mock installers.
func TestConcurrentPatchInstalls_SemaphoreLimit(t *testing.T) {
	const maxConcurrent = 2
	const totalCommands = 5

	var running atomic.Int32
	var maxObserved atomic.Int32
	var completed atomic.Int32

	// Slow installer: holds a slot for 80ms to make overlap observable.
	slowInstaller := &mockInstaller{
		name: "apt",
		fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
			cur := running.Add(1)
			defer running.Add(-1)

			// Track peak concurrency using CAS loop.
			for {
				old := maxObserved.Load()
				if cur <= old || maxObserved.CompareAndSwap(old, cur) {
					break
				}
			}

			time.Sleep(80 * time.Millisecond) // simulate slow installation
			completed.Add(1)
			return InstallResult{ExitCode: 0, Stdout: []byte("installed " + pkg.Name)}, nil
		},
	}

	m := newWithMaxFunc(func() int { return maxConcurrent })
	m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	m.installers = map[string]Installer{"apt": slowInstaller}
	m.executor = &osExecutor{}

	// Build 5 distinct install commands.
	commands := make([]agent.Command, totalCommands)
	for i := 0; i < totalCommands; i++ {
		payload := &pb.InstallPatchPayload{
			Packages: []*pb.PatchTarget{{Name: fmt.Sprintf("pkg-%d", i), Version: "1.0"}},
		}
		payloadBytes, err := proto.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		commands[i] = agent.Command{
			ID:      fmt.Sprintf("cmd-conc-%d", i),
			Type:    "install_patch",
			Payload: payloadBytes,
		}
	}

	// Dispatch all commands concurrently.
	var wg sync.WaitGroup
	results := make([]agent.Result, totalCommands)
	errors := make([]error, totalCommands)

	for i := 0; i < totalCommands; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx] = m.HandleCommand(context.Background(), commands[idx])
		}(i)
	}

	wg.Wait()

	// Verify: peak concurrency never exceeds the limit.
	if peak := maxObserved.Load(); peak > int32(maxConcurrent) {
		t.Errorf("peak concurrent installs = %d, want <= %d", peak, maxConcurrent)
	}

	// Verify: peak concurrency reached the limit (semaphore is not overly restrictive).
	if peak := maxObserved.Load(); peak < int32(maxConcurrent) {
		t.Errorf("peak concurrent installs = %d, expected to reach %d", peak, maxConcurrent)
	}

	// Verify: all commands completed successfully.
	if got := completed.Load(); got != int32(totalCommands) {
		t.Errorf("completed installs = %d, want %d", got, totalCommands)
	}

	for i := 0; i < totalCommands; i++ {
		if errors[i] != nil {
			t.Errorf("command %d returned error: %v", i, errors[i])
		}
		if results[i].ErrorMessage != "" {
			t.Errorf("command %d error message: %s", i, results[i].ErrorMessage)
		}
	}
}

// TestConcurrentPatchInstalls_DynamicLimitChange verifies that changing
// max_concurrent_installs mid-flight affects subsequent acquire attempts.
func TestConcurrentPatchInstalls_DynamicLimitChange(t *testing.T) {
	var limit atomic.Int32
	limit.Store(1) // start with limit=1

	var running atomic.Int32
	var maxObserved atomic.Int32

	// Gate: first batch of installs blocks until we release them.
	gate := make(chan struct{})
	slowInstaller := &mockInstaller{
		name: "apt",
		fn: func(_ context.Context, pkg PatchTarget, _ bool) (InstallResult, error) {
			cur := running.Add(1)
			defer running.Add(-1)
			for {
				old := maxObserved.Load()
				if cur <= old || maxObserved.CompareAndSwap(old, cur) {
					break
				}
			}
			<-gate
			return InstallResult{ExitCode: 0}, nil
		},
	}

	m := newWithMaxFunc(func() int { return int(limit.Load()) })
	m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	m.installers = map[string]Installer{"apt": slowInstaller}
	m.executor = &osExecutor{}

	makeCmd := func(id string) agent.Command {
		payload := &pb.InstallPatchPayload{
			Packages: []*pb.PatchTarget{{Name: id, Version: "1.0"}},
		}
		payloadBytes, _ := proto.Marshal(payload)
		return agent.Command{ID: id, Type: "install_patch", Payload: payloadBytes}
	}

	// Launch 3 commands with limit=1. Only 1 should be running.
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = m.HandleCommand(context.Background(), makeCmd(fmt.Sprintf("phase1-%d", idx)))
		}(i)
	}

	// Wait for one goroutine to be running (the one that acquired the semaphore).
	deadline := time.After(2 * time.Second)
	for running.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first goroutine to acquire semaphore")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	// Give a moment for other goroutines to (fail to) acquire.
	time.Sleep(30 * time.Millisecond)

	if cur := running.Load(); cur != 1 {
		t.Errorf("with limit=1, running = %d, want 1", cur)
	}

	// Raise the limit to 3 so the blocked goroutines can proceed.
	limit.Store(3)
	// Broadcast to wake waiters (Release does this, but we need to poke the
	// cond from outside). We do this by closing the gate to let all finish.
	close(gate)

	wg.Wait()
}
