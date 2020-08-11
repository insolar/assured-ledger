// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
)

func TestStartCtm(t *testing.T) {
	t.SkipNow()

	server := NewTestServer(t)
	defer server.Stop()

	// optional
	server.SetImposer(func(params *insconveyor.ImposedParams) {
		// impose per-test changes upon default behavior
	})
	server.Start()

	// do your test here
	server.NextPulse()

	// for i := 5; i > 0; i-- {
	// 	server.NextPulse()
	// 	time.Sleep(100*time.Millisecond)
	// }
	time.Sleep(10*time.Second)
}
