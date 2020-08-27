package terrors

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/monzo/terrors/stack"
)

type newError func(code, message string, params map[string]string) *Error

func TestLogParams(t *testing.T) {
	err := New("service.foo", "Some message", map[string]string{"public": "value"})

	assert.Equal(t, "value", err.LogMetadata()["public"])
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
	assert.True(t, err.PrefixMatches(ErrBadRequest, "missing_param"))
	assert.False(t, err.PrefixMatches(ErrInternalService))
	assert.False(t, err.PrefixMatches(ErrBadRequest+".missing_param.foo1"))
	assert.False(t, err.PrefixMatches(ErrBadRequest, "missing_param", "foo1"))
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
	assert.True(t, PrefixMatches(err, ErrBadRequest, "missing_param"))
	assert.False(t, PrefixMatches(err, ErrInternalService))
	assert.False(t, PrefixMatches(err, ErrBadRequest+".missing_param.foo1"))
	assert.False(t, PrefixMatches(err, ErrBadRequest, "missing_param", "foo1"))
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

func TestAugmentError(t *testing.T) {
	newErr := Augment(assert.AnError, "added context", map[string]string{
		"meta": "data",
	})
	terr := newErr.(*Error)
	assert.Equal(t, "internal_service", terr.Code)
	assert.Equal(t, "added context", terr.Message)

	assert.Equal(t, "internal_service: added context: assert.AnError general error for testing", terr.Error())
	assert.Equal(t, "data", terr.Params["meta"])
	assert.Equal(t, assert.AnError, terr.cause)
}

func TestAugmentTerror(t *testing.T) {
	base := NotFound("foo", "failed to find foo", map[string]string{
		"base": "meta",
	})
	newErr := Augment(base, "added context", map[string]string{
		"new": "meta",
	})
	terr := newErr.(*Error)
	assert.Equal(t, "not_found.foo", terr.Code)
	assert.Equal(t, "added context", terr.Message)
	assert.Empty(t, terr.StackFrames)

	assert.Equal(t, "not_found.foo: added context: failed to find foo", terr.Error())
	assert.Equal(t, base, terr.cause)
}

func TestAugmentNil(t *testing.T) {
	assert.Nil(t, Augment(nil, "added context", map[string]string{
		"new": "meta",
	}))
}

func TestIsError(t *testing.T) {
	cases := []struct {
		desc          string
		errCreator    func() error
		code          []string
		expectedMatch bool
	}{
		{
			desc: "non-terror",
			errCreator: func() error {
				return assert.AnError
			},
			code:          []string{ErrInternalService},
			expectedMatch: false,
		},
		{
			desc: "simple wrapped go error",
			errCreator: func() error {
				return Augment(assert.AnError, "added context", map[string]string{
					"meta": "data",
				})
			},
			code:          []string{ErrInternalService},
			expectedMatch: true,
		},
		{
			desc: "non-wrapped terror",
			errCreator: func() error {
				return NotFound("foo", "bar", nil)
			},
			code:          []string{ErrNotFound},
			expectedMatch: true,
		},
		{
			desc: "single-wrapped terror Augmentd",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return Augment(base, "added context", nil)
			},
			code:          []string{ErrNotFound},
			expectedMatch: true,
		},
		{
			desc: "multi-wrapped terror Augmentd",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				next := Augment(base, "added context", nil)
				return Augment(next, "more context", nil)
			},
			code:          []string{ErrNotFound},
			expectedMatch: true,
		},
		{
			desc: "multiple code parts match",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return Augment(base, "added context", nil)
			},
			code:          []string{ErrNotFound, "foo"},
			expectedMatch: true,
		},
		{
			desc: "multiple code parts mismatch",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return Augment(base, "added context", nil)
			},
			code:          []string{ErrNotFound, "notfoo"},
			expectedMatch: false,
		},
		{
			desc: "created NewInternalWithCause",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return NewInternalWithCause(base, "added context", nil, "")
			},
			code:          []string{ErrNotFound},
			expectedMatch: true,
		},
		{
			desc: "created NewInternalWithCause wrong code",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return NewInternalWithCause(base, "added context", nil, "")
			},
			code:          []string{ErrForbidden},
			expectedMatch: false,
		},
		{
			desc: "created NewInternalWithCause with subcode",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return NewInternalWithCause(base, "added context", nil, "downstream")
			},
			code:          []string{ErrInternalService, "downstream"},
			expectedMatch: true,
		},
		{
			desc: "created NewInternalWithCause with subcode mismatch",
			errCreator: func() error {
				base := NotFound("foo", "bar", nil)
				return NewInternalWithCause(base, "added context", nil, "downstream")
			},
			code:          []string{ErrInternalService, "mismatch"},
			expectedMatch: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expectedMatch, Is(tc.errCreator(), tc.code...))
		})
	}
}

func TestNewInternalWithCauseStack(t *testing.T) {
	err := NewInternalWithCause(assert.AnError, "test", nil, "")
	// Ensure that the first callsite is this method rather than the terrors internals
	assert.Contains(t, err.StackFrames[0].Method, "TestNewInternalWithCauseStack")
}

func TestPropagate(t *testing.T) {
	t.Run("terror", func(t *testing.T) {
		terr := &Error{Code: "foo"}
		out := Propagate(terr)
		assert.Equal(t, terr, out)
	})
	t.Run("non-terror", func(t *testing.T) {
		out := Propagate(assert.AnError)
		assert.IsType(t, &Error{}, out)
		terr := out.(*Error)
		assert.Equal(t, ErrInternalService, terr.Code)
		assert.Equal(t, assert.AnError, terr.cause)
		assert.Equal(t, assert.AnError.Error(), terr.Message)
		assert.Greater(t, len(terr.StackFrames), 0)
	})
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, Propagate(nil))
	})
}

func TestStackTrace(t *testing.T) {
	t.Run("nil stack", func(t *testing.T) {
		terr := &Error{}
		res := terr.StackTrace()
		assert.Len(t, res, 0)
	})
	t.Run("non-nil stack", func(t *testing.T) {
		terr := InternalService("foo", "bar", nil)
		res := terr.StackTrace()
		// Don't assert on content because it changes
		assert.NotEmpty(t, res)
	})
}
