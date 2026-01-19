package terrors_test

import (
	_ "embed"
	"encoding/json"
	"errors"
	pe "github.com/monzo/terrors/proto"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/monzo/terrors"
)

// there are two main cases here.
//
// a) we have an error in a child goroutine and handle it in a parent (eg: from a waitgroup)
// b) we have an error in a different process (eg: via RPC) and it gets propagated to the child
func TestStackTraceCrossGoRoutinePropagation(t *testing.T) {
	t.Skip("TODO")
	err := errorCanaryParentA()

	var terr *terrors.Error
	require.True(t, errors.As(err, &terr))

	stack := terr.StackString()
	t.Logf("Stack: %s", stack)
	assert.Contains(t, stack, "errorCanaryChild")
	assert.Contains(t, stack, "errorCanaryParentA")
}

func TestStackTraceCrossProcessPropagation(t *testing.T) {
	t.Skip("TODO")
	err := rpcCaller()

	var terr *terrors.Error
	require.True(t, errors.As(err, &terr))

	stack := terr.StackString()
	t.Logf("Stack: %s", stack)
	assert.Contains(t, stack, "rpcCaller")
	assert.Contains(t, stack, "rpcCallee")
	assert.Contains(t, stack, "rpcHandler")

	t.Skip("TODO")
}

func errorCanaryParentA() error {
	// Simulate a handler that uses a wait-group or similar
	err := errorCanaryChild()

	return terrors.Augment(err, "a", nil)
}

func errorCanaryChild() error {
	var err error

	// This is relatively common, and shows that wait-groups can often result in stack traces that aren't entirely helpful
	g := sync.WaitGroup{}
	g.Add(1)
	go func() {
		err = terrors.InternalService("test", "test", nil)
		g.Done()
	}()
	g.Wait()

	return err
}

//go:embed stack_propagation_sample.json
var rpcError []byte

func rpcCaller() error {
	var representation *pe.Error
	jsonErr := json.Unmarshal(rpcError, &representation)
	if jsonErr != nil {
		panic(jsonErr)
	}
	err := terrors.Unmarshal(representation)
	return terrors.Augment(err, "calling callee", nil)
}
