package lockstep_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/dcaiafa/lockstep"
)

func TestLockStep_EmitFirst(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(t)

	var state atomic.Int32

	go func() {
		time.Sleep(100 * time.Millisecond)

		state.Store(0)
		ls.Emit("e0")
		ls.Wait("w0")

		state.Store(1)
		ls.Emit("e1")
		ls.Wait("w1")

		state.Store(2)
		ls.Emit("done")
	}()

	ls.Wait("e0")
	expectEqual(t, 0, state.Load())
	ls.Emit("w0")

	ls.Wait("e1")
	expectEqual(t, 1, state.Load())
	ls.Emit("w1")

	ls.Wait("done")
	expectEqual(t, 2, state.Load())
}

func TestLockStep_WaitFirst(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(t)

	var state atomic.Int32

	go func() {
		state.Store(0)
		ls.Emit("e0")
		ls.Wait("w0")

		state.Store(1)
		ls.Emit("e1")
		ls.Wait("w1")

		state.Store(2)
		ls.Emit("done")
	}()

	time.Sleep(100 * time.Millisecond)

	ls.Wait("e0")
	expectEqual(t, 0, state.Load())
	ls.Emit("w0")

	ls.Wait("e1")
	expectEqual(t, 1, state.Load())
	ls.Emit("w1")

	ls.Wait("done")
	expectEqual(t, 2, state.Load())
}

func TestLockStep_MultiWait(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(t)

	go func() {
		ls.Emit("x")
		ls.Emit("z")
		ls.Emit("y")
	}()

	ls.Wait("x", "y", "z")
}

func TestLockStep_EmitTimeout(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(&PanicFailer{T: t})
	ls.SetTimeout(100 * time.Millisecond)

	expectFail(t, func() {
		ls.Emit("x")
	})
}

func TestLockStep_WaitTimeout(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(&PanicFailer{T: t})
	ls.SetTimeout(100 * time.Millisecond)

	expectFail(t, func() {
		ls.Wait("x")
	})
}

func TestLockStep_MultiWaitTimeout(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(&PanicFailer{T: t})
	ls.SetTimeout(100 * time.Millisecond)

	go func() {
		ls.Emit("x")
		ls.Emit("z")
	}()

	expectFail(t, func() {
		ls.Wait("x", "y", "z")
	})
}

func TestExample(t *testing.T) {
	t.Parallel()

	ls := lockstep.New(t)

	const d = 300 * time.Millisecond
	go func() {
		ls.Wait("go1")
		time.AfterFunc(d, func() {
			ls.Emit("done1")
		})
	}()
	go func() {
		ls.Wait("go2")
		<-time.After(d)
		ls.Emit("done2")
	}()

	begin := time.Now()
	ls.Emit("go1")
	ls.Emit("go2")
	ls.Wait("done1", "done2")
	dur := time.Since(begin)

	delta := dur - d
	if delta < 0 {
		delta *= -1
	}
	const e = 50 * time.Millisecond
	if delta > e {
		t.Fatalf("Expected callback in %v, actual was %v", d, dur)
	}
}
