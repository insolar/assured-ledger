package testutils

import (
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/reflectkit"
)

func CmpStateFuncs(want, got interface{}) bool {
	return reflectkit.CodeOf(want) == reflectkit.CodeOf(got)
}

type Tester interface {
	Helper()
	Errorf(format string, args ...interface{})
	FailNow()
}

// WaitSignalsTimed waits with timeout for signal channels to be closed
// or signaled with a message once. Waiting is ordered and not parallel,
// this may be beneficial in some situation and hurtful in others.
func WaitSignalsTimed(t Tester, timeout time.Duration, signals ...<-chan struct{}) {
	t.Helper()

	cleanUp := make(chan struct{})
	defer close(cleanUp)

	allGood := make(chan struct{})
	go func() {
		defer close(allGood)

		for _, signal := range signals {
			select {
			case <-signal:
			case <-cleanUp:
			}
		}
	}()

	select {
	case <-allGood:
	case <-time.After(timeout):
		require.FailNow(t, "timeout")
	}
}
