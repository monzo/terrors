package terrors

import (
	"errors"
	"testing"
)

// newTerrorAtDepth recurses to a given depth before constructing a fresh
// terror, so the benchmark captures (and symbolises) a stack of a realistic
// size rather than the shallow one the test harness itself provides.
func newTerrorAtDepth(depth int) *Error {
	if depth > 0 {
		return newTerrorAtDepth(depth - 1)
	}
	return New("benchmark.error", "something went wrong", map[string]string{
		"context": "benchmark",
	})
}

// augmentAtDepth recurses to a given depth before augmenting the given error,
// so the augment path builds a stack of a realistic size.
func augmentAtDepth(err error, depth int) error {
	if depth > 0 {
		return augmentAtDepth(err, depth-1)
	}
	return Augment(err, "additional context", map[string]string{
		"augment": "benchmark",
	})
}

// BenchmarkNew measures constructing a brand new terror from a realistic call
// depth. This exercises the stack capture and symbolisation on the hot path.
func BenchmarkNew(b *testing.B) {
	var err *Error
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err = newTerrorAtDepth(20)
	}
	_ = err
}

// BenchmarkAugmentGoError measures augmenting a vanilla Go error into a terror.
// This goes through NewInternalWithCause and captures a fresh stack.
func BenchmarkAugmentGoError(b *testing.B) {
	goErr := errors.New("plain go error")
	var err error
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err = augmentAtDepth(goErr, 20)
	}
	_ = err
}

// BenchmarkAugmentTerror measures augmenting an existing terror. This path
// captures the current stack to decide whether it shares a common ancestry
// with the stack already recorded on the terror being augmented.
func BenchmarkAugmentTerror(b *testing.B) {
	terr := New("benchmark.error", "something went wrong", nil)
	var err error
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err = augmentAtDepth(terr, 20)
	}
	_ = err
}
