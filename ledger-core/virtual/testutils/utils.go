// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/reflectkit"
)

func CmpStateFuncs(want, got interface{}) bool {
	return reflectkit.CodeOf(want) == reflectkit.CodeOf(got)
}

// WaitSignalsTimed waits with timeout for signal channels to be closed
// or signaled with a message once. Waiting is ordered and not parallel,
// this may be beneficial in some situation and hurtful in others.
func WaitSignalsTimed(t *testing.T, timeout time.Duration, signals ...<-chan struct{}) {
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
