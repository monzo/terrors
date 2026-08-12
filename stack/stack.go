// totally stolen from https://github.com/stvp/rollbar/blob/master/stack.go
package stack

import (
	"fmt"
	"hash/crc32"
	"runtime"
	"strings"
	"sync"
)

var (
	knownFilePathPatterns []string = []string{
		"github.com/",
		"code.google.com/",
		"bitbucket.org/",
		"launchpad.net/",
	}

	// pcBufPool reuses the []uintptr scratch buffers that runtime.Callers writes
	// into. BuildStack runs on every error, so recycling these avoids an 800-byte
	// allocation per call.
	pcBufPool = sync.Pool{
		New: func() interface{} {
			b := make([]uintptr, 100)
			return &b
		},
	}

	// symCache memoises the symbolisation of a program counter. A PC always maps
	// to the same frame(s) for the life of the process (code doesn't move), so
	// resolving file/line/function names — the expensive part of building a stack
	// — only has to happen once per distinct call site. The cache is bounded by
	// the number of call sites in the binary, so it cannot grow without bound.
	//
	// We deliberately cache frames by value, not by pointer: BuildStack copies
	// them into a fresh backing array on every call, so a caller that mutates a
	// returned Frame only affects their own Stack and can never corrupt the
	// shared cache (Frame is all strings and ints, so a struct copy is fully
	// independent).
	symCache sync.Map // map[uintptr][]Frame
)

type Frame struct {
	Filename string  `json:"filename"`
	Method   string  `json:"method"`
	Line     int     `json:"lineno"`
	PC       uintptr `json:"pc"`
}

type Stack []*Frame

func BuildStack(skip int) Stack {
	// Look up to a maximum depth of 100, reusing a pooled buffer. The buffer is
	// only referenced by runtime.CallersFrames below, so it's safe to return it
	// to the pool once we've finished iterating within this function.
	bufp := pcBufPool.Get().(*[]uintptr)
	ret := *bufp
	defer pcBufPool.Put(bufp)

	// Note that indexes must be one higher when passed to Callers()
	// than they would be when passed to Caller()
	// see https://golang.org/pkg/runtime/#Caller
	index := runtime.Callers(skip+1, ret)
	if index == 0 {
		// We have no frames to report, skip must be too high
		return Stack{}
	}

	// Symbolise each PC, reusing cached results for call sites we've already
	// resolved. We copy the cached frame values into a backing array that this
	// call exclusively owns, so callers can freely mutate the returned frames
	// without affecting the cache or any other Stack. A PC can expand to more
	// than one frame when calls are inlined, so index is a lower bound.
	frames := make([]Frame, 0, index)
	for _, pc := range ret[:index] {
		frames = append(frames, framesForPC(pc)...)
	}

	// Now the backing array is stable we can hand out pointers into it.
	stack := make(Stack, len(frames))
	for i := range frames {
		stack[i] = &frames[i]
	}
	return stack
}

// framesForPC resolves a single program counter to its frame(s), memoising the
// result. Symbolisation (decoding the runtime's line tables) is the dominant
// cost of building a stack, and the mapping is stable for the life of the
// process, so we only pay it once per distinct PC.
//
// The returned slice is the shared, cached copy and must not be mutated;
// callers append it into a private backing array (see BuildStack).
func framesForPC(pc uintptr) []Frame {
	if cached, ok := symCache.Load(pc); ok {
		return cached.([]Frame)
	}

	// A single PC can expand into multiple frames due to inlining, so we still
	// have to iterate CallersFrames. Processing one PC at a time yields the same
	// frames as processing the whole slice at once (verified in the tests).
	cf := runtime.CallersFrames([]uintptr{pc})
	var frames []Frame
	for {
		frame, more := cf.Next()
		frames = append(frames, Frame{
			Filename: shortenFilePath(frame.File),
			// frame.Function already carries the fully-qualified name, so we use
			// it directly rather than doing a second runtime.FuncForPC lookup.
			Method: functionName(frame.Function),
			Line:   frame.Line,
			PC:     frame.PC,
		})
		if !more {
			break
		}
	}

	// LoadOrStore keeps a single canonical slice per PC even if two goroutines
	// race to symbolise the same one concurrently.
	actual, _ := symCache.LoadOrStore(pc, frames)
	return actual.([]Frame)
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

func functionName(name string) string {
	if name == "" {
		return "???"
	}
	// Function names are always '/'-separated regardless of the host OS.
	end := strings.LastIndex(name, "/")
	return name[end+1:]
}
