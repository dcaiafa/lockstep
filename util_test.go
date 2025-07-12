package lockstep_test

import (
	"fmt"
	"testing"
)

type FailError string

func (e FailError) Error() string {
	return string(e)
}

type PanicFailer struct {
	*testing.T
}

func (f *PanicFailer) Fatalf(msg string, args ...any) {
	errMsg := fmt.Sprintf(msg, args...)
	panic(FailError(errMsg))
}

func expectFail(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		t.Helper()
		err := recover()
		if err == nil {
			t.Fatalf("Expected failure, but function succeeded")
		} else if _, ok := err.(FailError); !ok {
			panic(err)
		} else {
			// Test failure; just as expected.
		}
	}()

	f()
}

func expectEqual[T comparable](t *testing.T, e, a T) {
	t.Helper()
	if e != a {
		t.Fatalf("Expected: %v Actual: %v", e, a)
	}
}
