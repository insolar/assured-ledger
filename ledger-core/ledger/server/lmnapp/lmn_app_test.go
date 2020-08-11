// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
)

func TestStartCtm(t *testing.T) {
//	t.SkipNow()

	server := NewTestServer(t)
	defer server.Stop()

	// optional
	server.SetImposer(func(params *insconveyor.ImposedParams) {
		// impose per-test changes upon default behavior
	})
	server.Start()
	inject := server.Injector()

	var treeSvc treesvc.Service
	inject.MustInject(&treeSvc)

	// do your test here
	server.NextPulse()

	// genesis will run here and will initialize jet tree
	for {
		_, cur := treeSvc.GetTrees()
		if !cur.IsEmpty() {
			break
		}
		time.Sleep(10*time.Millisecond)
	}

//	server.NextPulse() 	// drops will be created

	// for i := 5; i > 0; i-- {
	// 	server.NextPulse()
	// 	time.Sleep(100*time.Millisecond)
	// }
	time.Sleep(time.Second)
}
