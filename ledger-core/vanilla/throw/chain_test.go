package throw

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type errType1 struct {
	m string
}

func (errType1) Error() string {
	return ""
}

type errType2 struct {
	m func() // incomparable
}

func (errType2) Error() string {
	return ""
}

func TestIsEqual(t *testing.T) {
	require.Panics(t, func() {
		var e1, e2 error = errType2{}, errType2{}
		runtime.KeepAlive(e1 == e2)
	})

	require.False(t, IsEqual(nil, errType1{}))
	require.False(t, IsEqual(errType1{}, nil))

	require.True(t, IsEqual(errType1{}, errType1{}))
	require.False(t, IsEqual(errType1{"A"}, errType1{"B"}))

	require.False(t, IsEqual(errType2{}, errType2{}))

	require.False(t, IsEqual(errType1{}, errType2{}))
	require.False(t, IsEqual(errType2{}, errType1{}))
}

type errBuilder struct {
	bottomErr error
}

func (v errBuilder) _err0() error {
	err := WithDetails(v.bottomErr, Unsupported())
	return err
}

func (v errBuilder) _err1() error {
	err := WithStackAndDetails(v._err0(), struct {
		msg string
		v0  int
	}{"err1Txt", 1})
	return err
}

func (v errBuilder) _err2() error {
	err := WithStack(v._err1())
	return err
}

func (v errBuilder) _err3() error {
	panic(v._err2())
}

func (v errBuilder) _err4() (err error) {
	defer func() {
		err = RW(recover(), err, "panicCatch", struct{ position int }{7})
	}()
	return v._err3()
}

func newChain(bottom error) error {
	return errBuilder{bottom}._err4()
}

func TestWrapPanicExt(t *testing.T) {
	err := WrapPanicExt("test", 0)
	st := OutermostStack(err).ShallowStackTrace()
	s := st.StackTraceAsText()
	methodName := "github.com/insolar/assured-ledger/ledger-core/vanilla/throw.TestWrapPanicExt"
	require.True(t, strings.HasPrefix(st.StackTraceAsText(), methodName), "missing method: %s", s)
}

func TestStackOf(t *testing.T) {
	errChain := newChain(io.EOF)
	st := DeepestStackTraceOf(errChain)
	require.NotNil(t, st)
	s := ErrorWithStack(errChain)
	methodName := "github.com/insolar/assured-ledger/ledger-core/vanilla/throw.TestStackOf"
	require.Contains(t, s, methodName)
}

func TestWalk(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		// Cannot use panics with value, because throw.stackTrace is not comparable
		require.Panics(t, func() {
			Walk(nil, nil)
		})
	})

	t.Run("empty error chain", func(t *testing.T) {
		require.False(t, Walk(nil, func(err error, holder StackTraceHolder) bool {
			return true
		}))
	})

	t.Run("simple error", func(t *testing.T) {
		require.True(t, Walk(fmt.Errorf("some error"), func(err error, holder StackTraceHolder) bool {
			return true
		}))
	})

	t.Run("single error chain", func(t *testing.T) {
		require.True(t, Walk(IllegalValue(), func(err error, holder StackTraceHolder) bool {
			return true
		}))
	})

	t.Run("double error chain", func(t *testing.T) {
		require.True(t, Walk(W(IllegalValue(), "one layer more", ""), func(err error, holder StackTraceHolder) bool {
			return true
		}))
	})
}

func TestPrintTo(t *testing.T) {
	t.Run("nil err with stack", func(t *testing.T) {
		b := strings.Builder{}
		PrintTo(nil, true, &b)
		require.Equal(t, "", b.String())
	})

	t.Run("nil err without stack", func(t *testing.T) {
		b := strings.Builder{}
		PrintTo(nil, false, &b)
		require.Equal(t, "", b.String())
	})

	t.Run("simple error without stack", func(t *testing.T) {
		b := strings.Builder{}
		PrintTo(fmt.Errorf("simple error"), false, &b)
		require.Equal(t, "simple error\n", b.String())
	})

	t.Run("simple error with stack", func(t *testing.T) {
		b := strings.Builder{}
		PrintTo(fmt.Errorf("simple error"), false, &b)
		require.Equal(t, "simple error\n", b.String())
	})

	t.Run("stacked error without stack", func(t *testing.T) {
		b := strings.Builder{}
		PrintTo(IllegalValue(), false, &b)
		require.Equal(t, "illegal value\n", b.String())
	})

	t.Run("stacked error with stack", func(t *testing.T) {
		b := strings.Builder{}
		PrintTo(IllegalValue(), true, &b)
		_, fileName, fileLine, ok := runtime.Caller(0)
		require.True(t, ok, "failed to get caller")
		require.Equal(t, fmt.Sprintf("illegal value\nStack trace:\ngithub.com/insolar/assured-ledger/ledger-core/vanilla/throw.TestPrintTo.func6\n\t%s:%d\n", fileName, fileLine-1), b.String())
	})
}
