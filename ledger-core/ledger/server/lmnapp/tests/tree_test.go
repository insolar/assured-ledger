// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp/lmntestapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
)

func TestGenesisTree(t *testing.T) {
	server := lmntestapp.NewTestServer(t)
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
	<-ch

	// but the jet tree is not available till pulse change
	prev, cur, ok := treeSvc.GetTrees(server.LastPulseNumber())
	require.True(t, ok)
	require.True(t, prev.IsEmpty())
	require.True(t, cur.IsEmpty())

	ch = jrn.WaitStopOf(&datawriter.SMPlash{}, 1)
	ch2 := jrn.WaitInitOf(&datawriter.SMDropBuilder{}, 1<<datawriter.DefaultGenesisSplitDepth)

	server.IncrementPulse() // tree will switch and drops will be created

	// but the jet tree is not available till pulse change
	prev, cur, ok = treeSvc.GetTrees(server.LastPulseNumber())
	require.True(t, ok)
	require.True(t, prev.IsEmpty())
	require.False(t, cur.IsEmpty())

	time.Sleep(time.Second / 4)

	<-ch
	<-ch2
}

func TestReadyTree(t *testing.T) {
	server := lmntestapp.NewTestServer(t)
	defer server.Stop()

	jrn := journal.New()

	var treeSvc treesvc.Service = treesvc.NewPerfect(
		datawriter.DefaultGenesisSplitDepth,
		server.Pulsar().GetLastPulseData().PulseNumber)

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		// impose per-test changes upon default behavior
		params.EventJournal = jrn

		deps := params.CompartmentSetup.Dependencies
		deps.ReplaceInterfaceDependency(&treeSvc)
	})

	ch := jrn.WaitStopOf(&datawriter.SMPlash{}, 1)

	server.Start()
	server.IncrementPulse() // trigger plash creation, but it will stop without a prev pulse
	<-ch

	// but the jet tree is already available
	prev, cur, ok := treeSvc.GetTrees(server.LastPulseNumber())
	require.True(t, ok)
	require.False(t, prev.IsEmpty())
	require.False(t, cur.IsEmpty())

	ch = jrn.WaitOnce(predicate.AfterStepName("(*SMPlash).stepWaitPast"))
	ch2 := jrn.WaitInitOf(&datawriter.SMDropBuilder{}, 1<<datawriter.DefaultGenesisSplitDepth)
	server.IncrementPulse() // trigger plash creation
	<-ch
	<-ch2
}

func TestRunGenesis(t *testing.T) {
	server := lmntestapp.NewTestServer(t)
	defer server.Stop()

	jrn := journal.New()

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		params.EventJournal = jrn
	})

	server.Start()
	server.RunGenesis()
	ch := jrn.WaitInitOf(&datawriter.SMDropBuilder{}, 1<<datawriter.DefaultGenesisSplitDepth)
	server.IncrementPulse()

	<-ch
}
