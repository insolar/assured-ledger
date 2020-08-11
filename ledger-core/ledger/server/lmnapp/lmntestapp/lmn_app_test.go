// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmntestapp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
)

func TestGenesisTree(t *testing.T) {
	server := NewTestServer(t)
	defer server.Stop()

	jrn := journal.New()
	// jrn.StartRecording(1000, true)

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		// impose per-test changes upon default behavior
		params.EventJournal = jrn
	})
	server.Start()
	inject := server.Injector()

	// do your test here

	var treeSvc treesvc.Service
	inject.MustInject(&treeSvc)

	ch := jrn.WaitStopOf(&datawriter.SMGenesis{}, 1)

	server.IncrementPulse()

	// genesis will run here and will initialize jet tree
	<- ch

	pn := server.Pulsar().GetLastPulseData().PulseNumber
	prev, cur, ok := treeSvc.GetTrees(pn)
	require.True(t, ok)
	require.True(t, prev.IsEmpty())
	require.False(t, cur.IsEmpty())

	ch = jrn.WaitStopOf(&datawriter.SMPlash{}, 1)
	ch2 := jrn.WaitInitOf(&datawriter.SMDropBuilder{}, 1<<datawriter.DefaultGenesisSplitDepth)

	server.IncrementPulse() 	// drops will be created as genesis is finished

	<- ch
	<- ch2
}
