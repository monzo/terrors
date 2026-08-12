package stack

import (
	"sync"
	"testing"
)

// buildStackAtDepth recurses to a given depth before building a stack, so the
// benchmark exercises a stack of a realistic size rather than the shallow one
// the test harness itself provides.
func buildStackAtDepth(depth int) Stack {
	if depth > 0 {
		return buildStackAtDepth(depth - 1)
	}
	return BuildStack(0)
}

// BenchmarkBuildStack measures the steady state: repeated calls from call sites
// that have already been symbolised, which is what a busy service overwhelmingly
// sees once its hot error paths have run at least once.
func BenchmarkBuildStack(b *testing.B) {
	var s Stack
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s = buildStackAtDepth(20)
	}
	_ = s
}

