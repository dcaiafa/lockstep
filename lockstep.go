// Package lockstep provides [Lockstep], a primitive for facilitating the
// testing of complex concurrent systems.
//
// [LockStep] supports two operations: Emit and Wait. An Emit operation with a
// message x will block until a corresponding Wait operation with message x is
// processed. Likewise, a Wait operation with y will also block until the
// corresponding Emit operation with y is processed.
//
// Example:
//
//   func TestExample(t *testing.T) {
//   	ls := lockstep.New(t)
//
//   	const d = 300 * time.Millisecond
//   	go func() {
//   		ls.Wait("go1")
//   		time.AfterFunc(d, func() {
//   			ls.Emit("done1")
//   		})
//   	}()
//   	go func() {
//   		ls.Wait("go2")
//   		<-time.After(d)
//   		ls.Emit("done2")
//   	}()
//
//   	begin := time.Now()
//   	ls.Emit("go1")
//   	ls.Emit("go2")
//   	ls.Wait("done1", "done2")
//   	dur := time.Since(begin)
//
//   	delta := dur - d
//   	if delta < 0 {
//   		delta *= -1
//   	}
//   	const e = 50 * time.Millisecond
//   	if delta > e {
//   		t.Fatalf("Expected callback in %v, actual was %v", d, dur)
//   	}
//   }

package lockstep

import (
	"context"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const DefaultTimeout = 10 * time.Second

// Lockstep is a testing primitive.
type LockStep struct {
	t       testing.TB
	verbose bool
	timeout time.Duration

	mu      sync.Mutex
	cv      *sync.Cond
	waiting map[string]bool
}

// New creates a LockStep instance. The provided test context will be used for
// logging and for timeout failures.
func New(t testing.TB) *LockStep {
	l := &LockStep{
		t:       t,
		timeout: DefaultTimeout,
		waiting: make(map[string]bool),
	}

	l.cv = sync.NewCond(&l.mu)

	return l
}

// SetTimeout overrides [DefaultTimeout] for Emit and Wait operations. Increase
// the timeout when debugging.
func (l *LockStep) SetTimeout(d time.Duration) {
	l.timeout = d
}

// SetVerbose configures verbose mode. If enabled, LockStep will emit detailed
// logs using t.Logf. Useful for debugging.
func (l *LockStep) SetVerbose(v bool) {
	l.verbose = v
}

// Emit will emit the message m. It will block until a corresponding Wait
// operation for m is processed.
func (l *LockStep) Emit(m string) {
	l.t.Helper()

	l.logf("Emiting %v", m)

	l.mu.Lock()
	defer l.mu.Unlock()

	deadline := time.Now().Add(l.timeout)
	for {
		if l.waiting[m] {
			l.logf("Emitted %v", m)
			delete(l.waiting, m)
			l.cv.Broadcast()
			return
		}

		if !l.waitWithLock(deadline) {
			l.t.Fatalf("Timeout emitting %v", m)
		}
	}
}

// Wait waits for all the provided messages. It will block until Emit operations
// corresponding to all messages have been processed.
//
// The order of Emit operations does not matter.
//
//	ls.Wait("x", "y")
//
// This Wait will be fulfilled if x and y are emitted in any order. Conversely:
//
//	ls.Wait("x")
//	ls.Wait("y")
//
// This Wait will only be fulfilled if x and y are emitted in order.
func (l *LockStep) Wait(ms ...string) {
	l.t.Helper()

	l.logf("Waiting for %v", messageList(slices.Values(ms)))

	waiting := make(map[string]bool, len(ms))

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, m := range ms {
		if l.waiting[m] {
			l.t.Fatalf("Double wait for %v", m)
		}
		l.waiting[m] = true
		waiting[m] = true
	}

	l.cv.Broadcast()

	deadline := time.Now().Add(l.timeout)
	for {
		for m := range waiting {
			if !l.waiting[m] {
				l.logf("Wait satisfied for %v", m)
				delete(waiting, m)
				l.cv.Broadcast()
			}
		}

		if len(waiting) == 0 {
			break
		}

		if !l.waitWithLock(deadline) {
			l.t.Fatalf("Timeout waiting for %v", messageList(maps.Keys(waiting)))
		}
	}
}

func (l *LockStep) waitWithLock(deadline time.Time) bool {
	l.t.Helper()

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	var timedOut atomic.Bool
	go func() {
		<-ctx.Done()
		l.cv.Broadcast()
		timedOut.Store(true)
	}()

	l.cv.Wait()

	return !timedOut.Load()
}

func (l *LockStep) logf(msg string, args ...any) {
	if l.verbose {
		l.t.Logf(msg, args...)
	}
}

func messageList(ms iter.Seq[string]) string {
	k := slices.Collect(ms)
	slices.Sort(k)
	return strings.Join(k, ", ")
}
