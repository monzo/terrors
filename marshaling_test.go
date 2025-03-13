package terrors

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pe "github.com/monzo/terrors/proto"
	"github.com/monzo/terrors/stack"
)

func TestMarshalNilError(t *testing.T) {
	var input *Error // nil
	protoError := Marshal(input)

	assert.NotNil(t, protoError)
	assert.Equal(t, ErrUnknown, protoError.Code)
	assert.NotEmpty(t, protoError.Message)
}

func TestUnmarshalNilError(t *testing.T) {
	var input *pe.Error // nil
	platError := Unmarshal(input)

	assert.NotNil(t, platError)
	assert.Equal(t, ErrUnknown, platError.Code)
	assert.Equal(t, "Nil error unmarshalled!", platError.Message)
}

// marshalTestCases represents a set of error formats
// which should be marshaled
var marshalTestCases = []struct {
	platErr  *Error
	protoErr *pe.Error
}{
	// confirm blank errors (shouldn't be possible) are UNKNOWN
	{
		&Error{},
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	// normal cases
	{
		&Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			StackFrames: []*stack.Frame{
				&stack.Frame{Filename: "some file", Method: "someMethod", Line: 123},
				&stack.Frame{Filename: "another file", Method: "someOtherMethod", Line: 1},
			},
		},
		&pe.Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			Stack: []*pe.StackFrame{
				{
					Filename: "some file",
					Line:     123,
					Method:   "someMethod",
				},
				{
					Filename: "another file",
					Line:     1,
					Method:   "someOtherMethod",
				},
			},
		},
	},
	{
		&Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
		},
		&pe.Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
		},
	},
	{
		&Error{
			Code:        ErrInternalService,
			Message:     "foo",
			IsRetryable: &notRetryable,
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Retryable: &pe.BoolValue{
				Value: false,
			},
		},
	},
	{
		&Error{
			Code:        ErrInternalService,
			Message:     "foo",
			IsRetryable: &retryable,
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Retryable: &pe.BoolValue{
				Value: true,
			},
		},
	},
	{
		&Error{
			Code:    ErrInternalService,
			Message: "foo",
		},
		&pe.Error{
			Code:      ErrInternalService,
			Message:   "foo",
			Retryable: nil,
		},
	},
	{
		&Error{
			Code:         ErrInternalService,
			Message:      "foo",
			IsUnexpected: &unexpected,
		},
		&pe.Error{
			Code:       ErrInternalService,
			Message:    "foo",
			Unexpected: &pe.BoolValue{Value: true},
		},
	},
	{
		&Error{
			Code:         ErrInternalService,
			Message:      "foo",
			IsUnexpected: &notUnexpected,
		},
		&pe.Error{
			Code:       ErrInternalService,
			Message:    "foo",
			Unexpected: &pe.BoolValue{Value: false},
		},
	},
	{
		&Error{
			Code:         ErrTimeout,
			Message:      "foo",
			IsRetryable:  &retryable,
			MarshalCount: 1000,
		},
		&pe.Error{
			Code:      ErrTimeout,
			Message:   "foo",
			Retryable: &pe.BoolValue{Value: true},
		},
	},
	{
		&Error{
			Code:         ErrTimeout,
			Message:      "5",
			MessageChain: []string{"4", "3", "2", "1"},
			IsRetryable:  &retryable,
			MarshalCount: 1000,
		},
		&pe.Error{
			Code:      ErrTimeout,
			Message:   "5",
			Retryable: &pe.BoolValue{Value: true},
		},
	},
}

func TestMarshal(t *testing.T) {
	for _, tc := range marshalTestCases {
		protoError := Marshal(tc.platErr)
		assert.Equal(t, tc.protoErr.Code, protoError.Code)
		assert.Equal(t, tc.protoErr.Message, protoError.Message)
		assert.Equal(t, tc.protoErr.Params, protoError.Params)
		assert.Equal(t, tc.platErr.MarshalCount+1, int(protoError.MarshalCount))

		if tc.platErr.IsRetryable == nil {
			assert.Nil(t, protoError.Retryable)
		} else {
			assert.Equal(t, *tc.platErr.IsRetryable, protoError.Retryable.Value)
			assert.Equal(t, tc.protoErr.Retryable.Value, protoError.Retryable.Value)
		}

		if tc.platErr.IsUnexpected == nil {
			assert.Nil(t, protoError.Unexpected)
		} else {
			assert.Equal(t, *tc.platErr.IsUnexpected, protoError.Unexpected.Value)
			assert.Equal(t, tc.protoErr.Unexpected.Value, protoError.Unexpected.Value)
		}

		if tc.platErr.MessageChain != nil {
			assert.Equal(t, tc.platErr.MessageChain, protoError.MessageChain)
		}
		t.Run("unmarshalling again keeps the same values", func(t *testing.T) {
			roundTripped := Unmarshal(protoError)
			// The `Code` can change (e.g. to "unknown") so we cannot assert on this
			assert.Equal(t, tc.platErr.Message, roundTripped.Message)
			assert.Equal(t, tc.platErr.MessageChain, roundTripped.MessageChain)

			assert.EqualValues(t, tc.platErr.IsRetryable, roundTripped.IsRetryable)
			assert.EqualValues(t, tc.platErr.IsUnexpected, roundTripped.IsUnexpected)

			// The marshal count is incremented by 1 on every marshal
			assert.Equal(t, tc.platErr.MarshalCount+1, roundTripped.MarshalCount)
		})
	}
}

// these are separate from above because the marshaling and unmarshaling isn't symmetric.
// protobuf turns empty maps[string]string into nil :(
var unmarshalTestCases = []struct {
	platErr  *Error
	protoErr *pe.Error
}{
	{
		New("", "", nil),
		&pe.Error{},
	},
	{
		New("", "", nil),
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	{
		&Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			StackFrames: []*stack.Frame{
				&stack.Frame{Filename: "some file", Method: "someMethod", Line: 123},
				&stack.Frame{Filename: "another file", Method: "someOtherMethod", Line: 1},
			},
		},
		&pe.Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			Stack: []*pe.StackFrame{
				{
					Filename: "some file",
					Line:     123,
					Method:   "someMethod",
				},
				{
					Filename: "another file",
					Line:     1,
					Method:   "someOtherMethod",
				},
			},
		},
	},
	{
		&Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
			Params:  map[string]string{},
		},
		&pe.Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
		},
	},
	{
		&Error{
			Code:        ErrInternalService,
			Message:     "foo",
			IsRetryable: &notRetryable,
			Params:      map[string]string{},
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Retryable: &pe.BoolValue{
				Value: false,
			},
		},
	},
	{
		&Error{
			Code:         ErrInternalService,
			Message:      "foo",
			IsUnexpected: &unexpected,
			Params:       map[string]string{},
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Unexpected: &pe.BoolValue{
				Value: true,
			},
		},
	},
	{
		&Error{
			Code:         ErrInternalService,
			Message:      "foo",
			IsUnexpected: &notUnexpected,
			Params:       map[string]string{},
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Unexpected: &pe.BoolValue{
				Value: false,
			},
		},
	},
	{
		&Error{
			Code:        ErrInternalService,
			Message:     "foo",
			IsRetryable: &retryable,
			Params:      map[string]string{},
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Retryable: &pe.BoolValue{
				Value: true,
			},
		},
	},
	{
		&Error{
			Code:        ErrInternalService,
			Message:     "foo",
			Params:      map[string]string{},
			IsRetryable: nil,
		},
		&pe.Error{
			Code:      ErrInternalService,
			Message:   "foo",
			Retryable: nil,
		},
	},
	{
		&Error{
			Code:         ErrInternalService,
			Message:      "foo",
			Params:       map[string]string{},
			IsRetryable:  &retryable,
			MarshalCount: 9876,
		},
		&pe.Error{
			Code:    ErrInternalService,
			Message: "foo",
			Retryable: &pe.BoolValue{
				Value: true,
			},
			MarshalCount: 9876,
		},
	},
	{
		&Error{
			Code:         ErrInternalService,
			Message:      "5",
			MessageChain: []string{"4", "3", "2", "1"},
			Params:       map[string]string{},
			IsRetryable:  &retryable,
			MarshalCount: 9876,
		},
		&pe.Error{
			Code:         ErrInternalService,
			Message:      "5",
			MessageChain: []string{"4", "3", "2", "1"},
			Retryable: &pe.BoolValue{
				Value: true,
			},
			MarshalCount: 9876,
		},
	},
}

func TestUnmarshal(t *testing.T) {
	for _, tc := range unmarshalTestCases {
		platErr := Unmarshal(tc.protoErr)
		assert.Equal(t, tc.platErr.Code, platErr.Code)
		assert.Equal(t, tc.platErr.Message, platErr.Message)
		assert.Equal(t, tc.platErr.Params, platErr.Params)
		assert.Equal(t, tc.platErr.MarshalCount, platErr.MarshalCount)

		if tc.platErr.IsRetryable == nil {
			assert.Nil(t, platErr.IsRetryable)
		} else {
			assert.Equal(t, *tc.platErr.IsRetryable, *platErr.IsRetryable)
		}

		assert.Equal(t, tc.platErr.Unexpected(), platErr.Unexpected())
		if tc.protoErr.Unexpected == nil {
			assert.False(t, platErr.Unexpected())
		} else {
			assert.Equal(t, tc.protoErr.Unexpected.Value, platErr.Unexpected())
		}

		if tc.platErr.MessageChain != nil {
			assert.Equal(t, tc.platErr.MessageChain, platErr.MessageChain)
		}
	}
}
