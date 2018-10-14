package terrors

import (
	"fmt"
	"os"
	"testing"

	"github.com/monzo/terrors/stack"
	"github.com/stretchr/testify/assert"
)

type newError func(code, message string, params map[string]string) *Error

func TestLogParams(t *testing.T) {
	errors := []Error{
		{
			Code:        "service.foo",
			Message:     "Some message",
			Params:      map[string]string{"public": "value"},
			StackFrames: nil,
		},
		*New("service.foo", "Some message", map[string]string{"public": "value"}),
	}

	for _, err := range errors {
		assert.Equal(t, "value", err.LogMetadata()["public"])
		if err.StackFrames != nil {
			assert.Equal(t, "testing.tRunner", err.LogMetadata()["terrors_function"])
		} else {
			assert.Equal(t, err.Params, err.LogMetadata())
		}
	}
}

func TestErrorConstructors(t *testing.T) {

	testCases := []struct {
		constructor  newError
		code         string
		message      string
		params       map[string]string
		expectedCode string
	}{
		{
			InternalService, "service.foo", "internal_service.service.foo", nil, ErrInternalService,
		},
		{
			BadRequest, "service.foo", "bad_request.service.foo", nil, ErrBadRequest,
		},
		{
			BadResponse, "service.foo", "bad_response.service.foo", nil, ErrBadResponse,
		},
		{
			Timeout, "service.foo", "timeout.service.foo", nil, ErrTimeout,
		},
		{
			NotFound, "service.foo", "not_found.service.foo", nil, ErrNotFound,
		},
		{
			Forbidden, "service.foo", "forbidden.service.foo", nil, ErrForbidden,
		},
		{
			Unauthorized, "service.foo", "unauthorized.service.foo", nil, ErrUnauthorized,
		},
		{
			Unauthorized, "service.foo", "test params", map[string]string{
				"some key":    "some value",
				"another key": "another value",
			}, ErrUnauthorized,
		},
		{
			PreconditionFailed, "service.foo", "precondition_failed.service.foo", nil, ErrPreconditionFailed,
		},
	}

	for _, tc := range testCases {
		err := tc.constructor(tc.code, tc.message, tc.params)
		assert.Equal(t, fmt.Sprintf("%s.%s", tc.expectedCode, tc.code), err.Code)
		assert.Equal(t, fmt.Sprintf("%s: %s", err.Code, tc.message), err.Error())
		if len(tc.params) > 0 {
			assert.Equal(t, tc.params, err.Params)
		}

	}
}

func TestNew(t *testing.T) {
	err := New("service.foo", "Some message", map[string]string{
		"public": "value",
	})

	assert.Equal(t, "service.foo", err.Code)
	assert.Equal(t, "Some message", err.Message)
	assert.Equal(t, map[string]string{
		"public": "value",
	}, err.Params)
}

func TestWrapWithWrappedErr(t *testing.T) {
	err := &Error{
		Code:        ErrForbidden,
		Message:     "Some message",
		StackFrames: stack.BuildStack(0),
		Params: map[string]string{
			"something old": "caesar",
		},
	}

	wrappedErr := Wrap(err, map[string]string{
		"something new": "a computer",
	}).(*Error)

	assert.Equal(t, err.Code, wrappedErr.Code)
	assert.Equal(t, err.StackFrames, wrappedErr.StackFrames)
	assert.Equal(t, err.Message, wrappedErr.Message)
	assert.Equal(t, wrappedErr.Params, map[string]string{
		"something old": "caesar",
		"something new": "a computer",
	})

}

func TestWrap(t *testing.T) {
	err := fmt.Errorf("Look here, an error")
	wrappedErr := Wrap(err, map[string]string{
		"blub": "dub",
	}).(*Error)

	assert.Equal(t, "internal_service: Look here, an error", wrappedErr.Error())
	assert.Equal(t, "Look here, an error", wrappedErr.Message)
	assert.Equal(t, ErrInternalService, wrappedErr.Code)
	assert.Equal(t, wrappedErr.Params, map[string]string{
		"blub": "dub",
	})

}

func TestErrorString(t *testing.T) {
	tt := []struct {
		given    *Error
		expected string
	}{
		{
			nil,
			"",
		},
		{
			&Error{
				Code:    "",
				Message: "Error with no code",
			},
			"Error with no code",
		},
		{
			New("no_message", "", nil),
			"no_message",
		},
		{
			&Error{
				Code:    "",
				Message: "",
			},
			"",
		},
		{
			New("unknown", "error message", nil),
			"unknown: error message",
		},
	}
	for _, tc := range tt {
		assert.Equal(t, tc.expected, tc.given.Error())
	}
}

func getNilErr() error {
	return Wrap(nil, nil)
}

func TestNilError(t *testing.T) {
	assert.Equal(t, getNilErr(), nil)
	assert.Nil(t, getNilErr())
	assert.Nil(t, Wrap(nil, nil))
}

func TestMatchesMethod(t *testing.T) {
	err := &Error{
		Code:    "bad_request.missing_param.foo",
		Message: "You need to pass a value for foo; try passing foo=bar",
	}
	assert.True(t, err.Matches(ErrBadRequest))
	assert.True(t, err.Matches(ErrBadRequest+".missing_param"))
	assert.False(t, err.Matches(ErrInternalService))
	assert.False(t, err.Matches(ErrBadRequest+".missing_param.foo1"))
	assert.True(t, err.Matches("You need to pass a value for foo"))
}

func TestMatches(t *testing.T) {
	err := &Error{
		Code:    "bad_request.missing_param.foo",
		Message: "You need to pass a value for foo; try passing foo=bar",
	}
	assert.True(t, Matches(err, ErrBadRequest))
	assert.True(t, Matches(err, ErrBadRequest+".missing_param"))
	assert.False(t, Matches(err, ErrInternalService))
	assert.False(t, Matches(err, ErrBadRequest+".missing_param.foo1"))
	assert.True(t, Matches(err, "You need to pass a value for foo"))
	assert.False(t, Matches(nil, ErrBadRequest))
}

func TestPrefixMatchesMethod(t *testing.T) {
	err := &Error{
		Code:    "bad_request.missing_param.foo",
		Message: "You need to pass a value for foo; try passing foo=bar",
	}
	assert.True(t, err.PrefixMatches(ErrBadRequest))
	assert.True(t, err.PrefixMatches(ErrBadRequest+".missing_param"))
	assert.False(t, err.PrefixMatches(ErrInternalService))
	assert.False(t, err.PrefixMatches(ErrBadRequest+".missing_param.foo1"))
	assert.False(t, err.PrefixMatches("You need to pass a value for foo"))
	assert.False(t, err.PrefixMatches("missing_param"))
}

func TestPrefixMatches(t *testing.T) {
	err := &Error{
		Code:    "bad_request.missing_param.foo",
		Message: "You need to pass a value for foo; try passing foo=bar",
	}
	assert.True(t, PrefixMatches(err, ErrBadRequest))
	assert.True(t, PrefixMatches(err, ErrBadRequest+".missing_param"))
	assert.False(t, PrefixMatches(err, ErrInternalService))
	assert.False(t, PrefixMatches(err, ErrBadRequest+".missing_param.foo1"))
	assert.False(t, PrefixMatches(err, "You need to pass a value for foo"))
	assert.False(t, PrefixMatches(err, "missing_param"))
	assert.False(t, PrefixMatches(nil, ErrBadRequest))
}

func ExampleWrapWithCode() {
	fn := "not/a/file"
	_, err := os.Open(fn)
	if err != nil {
		errParams := map[string]string{
			"filename": fn,
		}
		err = WrapWithCode(err, errParams, ErrNotFound)
		terr := err.(*Error)
		fmt.Println(terr.Error())
		// Output: not_found: open not/a/file: no such file or directory
	}
}

func ExampleMatches() {
	err := NotFound("handler_missing", "Handler not found", nil)
	fmt.Println(Matches(err, "not_found.handler_missing"))
	// Output: true
}
