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
