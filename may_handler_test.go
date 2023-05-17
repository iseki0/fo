package fo

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestWithLoggerHandler(t *testing.T) {
	t.Parallel()

	logger := newMayTestLogger()

	loggerHandler := WithLoggerHandler(logger)
	loggerHandler(assert.AnError, "error occurred: %s", "foo")

	assert.Equal(t, "error occurred: foo: assert.AnError general error for testing\n", logger.sb.String())
}

func TestWithLogFuncHandler(t *testing.T) {
	t.Parallel()

	sb := strings.Builder{}

	printFunc := func(v ...any) {
		sb.WriteString(fmt.Sprintln(v...))
	}

	logFuncHandler := WithLogFuncHandler(printFunc)
	logFuncHandler(assert.AnError, "error occurred: %s", "foo")

	assert.Equal(t, "error occurred: foo: assert.AnError general error for testing\n", sb.String())
}

func TestMessageFromMsgAndArgs(t *testing.T) {
	t.Parallel()

	msg := messageFromMsgAndArgs()
	assert.Empty(t, msg)

	msg = messageFromMsgAndArgs("error occurred")
	assert.Equal(t, "error occurred", msg)

	msg = messageFromMsgAndArgs([]string{"error occurred", "foo"})
	assert.Equal(t, "[error occurred foo]", msg)

	msg = messageFromMsgAndArgs("error occurred: %s", "foo")
	assert.Equal(t, "error occurred: foo", msg)

	msg = messageFromMsgAndArgs([]string{"error occurred", "foo"}, "bar")
	assert.Equal(t, "[[error occurred foo] bar]", msg)
}

func TestFormatErrorWithMessageArgs(t *testing.T) {
	t.Parallel()

	err := formatErrorWithMessageArgs(nil)
	assert.NoError(t, err)

	err = formatErrorWithMessageArgs(errNotOK, "error occurred")
	require.Error(t, err)
	assert.EqualError(t, err, "error occurred")

	err = formatErrorWithMessageArgs(assert.AnError)
	require.Error(t, err)
	assert.EqualError(t, err, "assert.AnError general error for testing")

	err = formatErrorWithMessageArgs(assert.AnError, "error occurred")
	require.Error(t, err)
	assert.EqualError(t, err, "error occurred: assert.AnError general error for testing")

	err = formatErrorWithMessageArgs(assert.AnError, []string{"error occurred", "foo"})
	require.Error(t, err)
	assert.EqualError(t, err, "[error occurred foo]: assert.AnError general error for testing")
}

func TestFormatError(t *testing.T) {
	t.Parallel()

	err := formatError(nil)
	assert.NoError(t, err)

	err = formatError(true)
	assert.NoError(t, err)

	err = formatError(false)
	require.Error(t, err)
	assert.EqualError(t, err, "not ok")

	err = formatError(assert.AnError)
	require.Error(t, err)
	assert.EqualError(t, err, "assert.AnError general error for testing")

	assert.PanicsWithValue(t, "may: invalid err type '[]string', should either be a bool or an error", func() {
		_ = formatError([]string{"foo", "bar"})
	})
}

func TestMayHandlersHandleError(t *testing.T) {
	mayHandlers := newMayHandlers()

	mayHandlers.handlers = append(mayHandlers.handlers, func(err error, v ...any) {
		assert.EqualError(t, err, "assert.AnError general error for testing")
		assert.Equal(t, []any{"error occurred: %s", "foo"}, v)
	})

	mayHandlers.handleError(nil)
	mayHandlers.handleError(assert.AnError, "error occurred: %s", "foo")
}

func TestCollectAsError(t *testing.T) {
	may := NewMay[string]()

	may.Invoke("", errors.New("something went wrong"))
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail")
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail with %s", "foo")

	err := may.CollectAsError()
	assert.EqualError(t, err, "something went wrong; operation shouldn't fail: something went wrong; operation shouldn't fail with foo: something went wrong")
}

func TestCollectAsErrors(t *testing.T) {
	may := NewMay[string]()

	errs := may.CollectAsErrors()
	assert.Empty(t, errs)

	may.Invoke("", errors.New("something went wrong"))
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail")
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail with %s", "foo")

	errs = may.CollectAsErrors()
	assert.EqualError(t, errs[0], "something went wrong")
	assert.EqualError(t, errs[1], "operation shouldn't fail: something went wrong")
	assert.EqualError(t, errs[2], "operation shouldn't fail with foo: something went wrong")
}

func handleErrorTestFunc() (err error) {
	may := NewMay[string]()

	defer may.HandleErrors(func(errs []error) {
		err = fmt.Errorf("error occurred: %w", multierr.Combine(errs...))
	})

	may.Invoke("", errors.New("something went wrong"))
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail")
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail with %s", "foo")

	return nil
}

func TestHandleErrors(t *testing.T) {
	err := handleErrorTestFunc()
	assert.EqualError(t, err, "error occurred: something went wrong; operation shouldn't fail: something went wrong; operation shouldn't fail with foo: something went wrong")

	err = nil

	NewMay[string]().HandleErrors(func(errs []error) {
		if len(errs) > 0 {
			err = fmt.Errorf("error occurred: %w", multierr.Combine(errs...))
		}
	})
	assert.NoError(t, err)
}

func handleErrorWithReturnTestFunc() (err error) {
	may := NewMay[string]()

	defer func() {
		err = may.HandleErrorsWithReturn(func(errs []error) error {
			return fmt.Errorf("error occurred: %w", multierr.Combine(errs...))
		})
	}()

	may.Invoke("", errors.New("something went wrong"))
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail")
	may.Invoke("", errors.New("something went wrong"), "operation shouldn't fail with %s", "foo")

	return nil
}

func TestHandleErrorsWithReturn(t *testing.T) {
	err := handleErrorWithReturnTestFunc()
	assert.EqualError(t, err, "error occurred: something went wrong; operation shouldn't fail: something went wrong; operation shouldn't fail with foo: something went wrong")

	err = NewMay[string]().HandleErrorsWithReturn(func(errs []error) error {
		if len(errs) > 0 {
			return fmt.Errorf("error occurred: %w", multierr.Combine(errs...))
		}

		return nil
	})
	assert.NoError(t, err)
}
