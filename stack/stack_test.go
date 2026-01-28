// stolen from https://github.com/stvp/rollbar/blob/master/stack_test.go
package stack

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildStack(t *testing.T) {
	frame := BuildStack(1)[0]
	assert.Equal(t, "github.com/monzo/terrors/stack/stack_test.go", frame.Filename)
	assert.Equal(t, "stack.TestBuildStack", frame.Method)
	assert.NotZero(t, frame.Line, "Frame line number")
}

func TestStackFingerprint(t *testing.T) {
	tests := []struct {
		Fingerprint string
		Stack       Stack
	}{
		{
			"9344290d",
			Stack{
				&Frame{"foo.go", "Oops", 1, 0},
			},
		},
		{
			"a4d78b7",
			Stack{
				&Frame{"foo.go", "Oops", 2, 0},
			},
		},
		{
			"50e0fcb3",
			Stack{
				&Frame{"foo.go", "Oops", 1, 0},
				&Frame{"foo.go", "Oops", 2, 0},
			},
		},
	}

	for i, test := range tests {
		fingerprint := test.Stack.Fingerprint()
		if fingerprint != test.Fingerprint {
			t.Errorf("tests[%d]: got %s", i, fingerprint)
		}
	}
}

func TestShortenFilePath(t *testing.T) {
	tests := []struct {
		Given    string
		Expected string
	}{
		{"", ""},
		{"foo.go", "foo.go"},
		{"/usr/local/go/src/pkg/runtime/proc.c", "pkg/runtime/proc.c"},
		{"/home/foo/go/src/github.com/stvp/rollbar.go", "github.com/stvp/rollbar.go"},
	}
	for i, test := range tests {
		got := shortenFilePath(test.Given)
		if got != test.Expected {
			t.Errorf("tests[%d]: got %s", i, got)
		}
	}
}

func TestCommonAncestrySamePCShouldMatch(t *testing.T) {
	current := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}
	other := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}

	assertHasCommonAncestry(t, current, other)
}

func TestCommonAncestryDifferingRootPCShouldNotMatch(t *testing.T) {
	current := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}
	other := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 4, Method: "four"}}

	assertHasNoCommonAncestry(t, current, other)
}

func TestCommonAncestryDifferingPcInTopFrameButSameMethodShouldMatch(t *testing.T) {
	current := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}
	other := Stack{{PC: 4, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}

	assertHasCommonAncestry(t, current, other)
}

func TestCommonAncestrySamePCExtraFrameLeftShouldNotMatch(t *testing.T) {
	current := Stack{{PC: 2, Method: "two"}, {PC: 3, Method: "three"}}
	other := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}

	assertHasNoCommonAncestry(t, current, other)
}

func TestCommonAncestrySamePCExtraFrameRightShouldMatch(t *testing.T) {
	current := Stack{{PC: 1, Method: "one"}, {PC: 2, Method: "two"}, {PC: 3, Method: "three"}}
	other := Stack{{PC: 2, Method: "two"}, {PC: 3, Method: "three"}}

	assertHasCommonAncestry(t, current, other)
}

func assertHasCommonAncestry(t *testing.T, current Stack, other Stack) bool {
	return assert.True(t, current.HasCommonAncestry(other), "Stack current has common ancestry with other: current:%v\nother:%v\n", current, other)
}
func assertHasNoCommonAncestry(t *testing.T, current Stack, other Stack) bool {
	return assert.False(t, current.HasCommonAncestry(other), "Stack current should not have common ancestry with other: current:%v\nother:%v\n", current, other)
}
