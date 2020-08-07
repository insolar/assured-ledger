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

	for i := 3; i > 0; i-- {
		time.Sleep(10*time.Millisecond)
		server.NextPulse()
	}
}
