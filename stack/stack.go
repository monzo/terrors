// totally stolen from https://github.com/stvp/rollbar/blob/master/stack.go
package stack

import (
	"fmt"
	"hash/crc32"
	"os"
	"runtime"
	"strings"
)

var (
	knownFilePathPatterns []string = []string{
		"github.com/",
		"code.google.com/",
		"bitbucket.org/",
		"launchpad.net/",
	}
)

type Frame struct {
	Filename string  `json:"filename"`
	Method   string  `json:"method"`
	Line     int     `json:"lineno"`
	PC       uintptr `json:"pc"`
}

type Stack []*Frame

func BuildStack(skip int) Stack {
	stack := make(Stack, 0)

	// Look up to a maximum depth of 100
	ret := make([]uintptr, 100)

	// Note that indexes must be one higher when passed to Callers()
	// than they would be when passed to Caller()
	// see https://golang.org/pkg/runtime/#Caller
	index := runtime.Callers(skip+1, ret)
	if index == 0 {
		// We have no frames to report, skip must be too high
		return stack
	}

	// This function takes a list of counters and gets function/file/line information
	cf := runtime.CallersFrames(ret[:index])

	for {
		frame, ok := cf.Next()
		stack = append(stack, &Frame{
			Filename: shortenFilePath(frame.File),
			Method:   functionName(frame.PC),
			Line:     frame.Line,
			PC:       frame.PC,
		})
		if !ok {
			// This was the last valid caller
			break
		}
	}
	return stack
}

// Create a fingerprint that uniquely identify a given message. We use the full
// callstack, including file names. That ensure that there are no false
// duplicates but also means that after changing the code (adding/removing
// lines), the fingerprints will change. It's a trade-off.
func (s Stack) Fingerprint() string {
	hash := crc32.NewIEEE()
	for _, frame := range s {
		fmt.Fprintf(hash, "%s%s%d", frame.Filename, frame.Method, frame.Line)
	}
	return fmt.Sprintf("%x", hash.Sum32())
}

func (s Stack) HasCommonAncestry(otherRoot Stack) bool {
	startIdx := len(otherRoot) - len(s)
	if startIdx < 0 {
		return false
	}

	other := otherRoot[startIdx:]

	if len(other) == 0 {
		return true
	}

	// Special case the frame where we call terrors.Augment, because for cases like the following:
	//
	//   err := something();
	//   if err != nil {
	//   	return terrors.Augment(err, "context", nil()
	//   }
	//
	// Just comparing the program counter isn't enough, because while yes, the calls
	// to something() and terrors.Augment() are at different points, we still want to
	// consider them as the "same" stack frame. So we fall back to comparing the file
	// and method names too.
	if !topFrameEqualByFunctionName(other[0], s[0]) {
		return false
	}

	thisRemaining := s[1:]
	otherRemaining := other[1:]

	for i, thisFrame := range thisRemaining {
		if !equalByPC(otherRemaining[i], thisFrame) {
			return false
		}
	}
	return true
}

func topFrameEqualByFunctionName(otherFrame *Frame, thisFrame *Frame) bool {
	// We also assume that a program counter of zero means it is remote, and thus
	// never equal to a local frame, since a) we don't currently transfer that value
	// over the wire (see the protobuf representations for details), and b) there are
	// no instructions mapped at address zero.
	if otherFrame.PC == 0 {
		return false
	}

	if thisFrame.PC == otherFrame.PC {
		// The other properties are all derived from the program counter, so we know that
		// if the program counters are the same we can skip the remaining checks.
		return true
	} else if thisFrame.Filename == otherFrame.Filename && thisFrame.Method == otherFrame.Method {
		return true
	}
	return false
}

func equalByPC(otherFrame *Frame, thisFrame *Frame) bool {
	// A frame from a remote source; can't match a locally generated stack
	if otherFrame.PC == 0 {
		return false
	}

	// if the program counter values are the same, then that's fine, and all we need to care about for frames above the caller of terrors.Augment
	return thisFrame.PC == otherFrame.PC
}

func (s Stack) String() string {
	var buf strings.Builder
	s.WriteWithMaxSize(&buf, 32000)
	return buf.String()
}

// WriteWithMaxSize writes the stack to the provided buffer, not going above
func (s Stack) WriteWithMaxSize(buffer *strings.Builder, sizeLimit int) bool {
	for _, frame := range s {
		// 10 seems like a reasonable estimate of how large the rest of the line would be.
		estimatedLineLen := len(frame.Filename) + len(frame.Method) + 16
		if estimatedLineLen+buffer.Len() > sizeLimit {
			return true
		}
		fmt.Fprintf(buffer, "\n  %s:%d in %s", frame.Filename, frame.Line, frame.Method)
	}

	return false
}

// Remove un-needed information from the source file path. This makes them
// shorter in Rollbar UI as well as making them the same, regardless of the
// machine the code was compiled on.
//
// Examples:
//
//	/usr/local/go/src/pkg/runtime/proc.c -> pkg/runtime/proc.c
//	/home/foo/go/src/github.com/rollbar/rollbar.go -> github.com/rollbar/rollbar.go
func shortenFilePath(s string) string {
	idx := strings.Index(s, "/src/pkg/")
	if idx != -1 {
		return s[idx+5:]
	}
	for _, pattern := range knownFilePathPatterns {
		idx = strings.Index(s, pattern)
		if idx != -1 {
			return s[idx:]
		}
	}
	return s
}

func functionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}
	name := fn.Name()
	end := strings.LastIndex(name, string(os.PathSeparator))
	return name[end+1:]
}
