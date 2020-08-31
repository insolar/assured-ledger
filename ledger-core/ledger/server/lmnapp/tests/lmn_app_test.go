// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp/lmntestapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
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

	ch = jrn.WaitStopOf(&datawriter.SMPlash{}, 1)
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

func TestAddRecords(t *testing.T) {
	server := lmntestapp.NewTestServer(t)
	defer server.Stop()

	var recBuilder lmntestapp.RecordBuilder

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		recBuilder = lmntestapp.NewRecordBuilderFromDependencies(params.AppInject)
	})

	server.Start()
	server.RunGenesis()
	server.IncrementPulse()

	recBuilder.RefTemplate = reference.NewSelfRefTemplate(server.LastPulseNumber(), reference.SelfScopeLifeline)

	genNewLine := generatorNewLifeline{
		recBuilder: recBuilder,
		conv:       server.App().Conveyor(),
		body:       make([]byte, 1<<10),
	}

	reasonRef := gen.UniqueGlobalRefWithPulse(server.LastPulseNumber())

	var bundleSignatures [][]cryptkit.Signature

	t.Run("one bundle", func(t *testing.T) {
		for N := 10; N > 0; N-- {
			// one bundle per object
			sg, err := genNewLine.registerNewLine(reasonRef)
			require.NoError(t, err)
			require.Len(t, sg, 3)
			bundleSignatures = append(bundleSignatures, sg)
		}
	})

	t.Run("overlapped", func(t *testing.T) {
		for N := 10; N > 0; N-- {
			// two intersecting bundles per object
			fullSet := genNewLine.makeSet(reasonRef)
			firstSet := fullSet

			// first record only
			firstSet.Requests = firstSet.Requests[:1]
			sg, err := genNewLine.callRegister(firstSet)
			require.NoError(t, err)
			require.Len(t, sg, 1)
			sgCopy := sg[0]

			// all records together
			sg, err = genNewLine.callRegister(fullSet)
			require.NoError(t, err)
			require.Len(t, sg, 3)
			require.True(t, sgCopy.Equals(sg[0]))

			bundleSignatures = append(bundleSignatures, sg)
		}
	})

	// repeat the same sequence
	// all registrations must be ok as they will be deduplicated
	genNewLine.seqNo.Store(0)

	t.Run("duplicates", func(t *testing.T) {
		i := 0
		for N := 20; N > 0; N-- {
			sg, err := genNewLine.registerNewLine(reasonRef)
			require.NoError(t, err)
			require.Len(t, sg, 3)

			// make sure to get exactly same signatures - this is only possible when records were deduplicated
			for j := range sg {
				require.True(t, bundleSignatures[i][j].Equals(sg[j]), i)
			}

			i++
		}
	})
}

func BenchmarkWriteNew(b *testing.B) {
	b.Run("1k", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<10, false)
	})

	b.Run("1k-par", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<10, true)
	})

	b.Run("16k", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<14, false)
	})

	b.Run("16k-par", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<14, true)
	})

	b.Run("128k", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<17, false)
	})

	b.Run("128k-par", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<17, true)
	})

	b.Run("1M", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<20, false)
	})

	b.Run("1M-par", func(b *testing.B) {
		benchmarkWriteNew(b, 1<<20, true)
	})
}

func benchmarkWriteNew(b *testing.B, bodySize int, parallel bool) {
	server := lmntestapp.NewTestServer(b)
	defer server.Stop()

	var recBuilder lmntestapp.RecordBuilder

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		recBuilder = lmntestapp.NewRecordBuilderFromDependencies(params.AppInject)
	})

	server.Start()
	server.RunGenesis()
	server.IncrementPulse()

	recBuilder.RefTemplate = reference.NewSelfRefTemplate(server.LastPulseNumber(), reference.SelfScopeLifeline)

	genNewLine := generatorNewLifeline{
		recBuilder: recBuilder,
		conv:       server.App().Conveyor(),
		body:       make([]byte, bodySize),
	}

	reasonRef := gen.UniqueGlobalRefWithPulse(server.LastPulseNumber())

	b.ResetTimer()
	b.ReportAllocs()

	if parallel {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = genNewLine.registerNewLine(reasonRef)
			}
		})
	} else {
		for i := b.N; i > 0; i-- {
			_, _ = genNewLine.registerNewLine(reasonRef)
		}
	}

	b.SetBytes(int64(genNewLine.totalBytes.Load()) / int64(genNewLine.seqNo.Load()))
}

type generatorNewLifeline struct {
	recBuilder lmntestapp.RecordBuilder
	seqNo      atomickit.Uint32
	totalBytes atomickit.Uint64
	body       []byte
	conv       *conveyor.PulseConveyor
}

func (p *generatorNewLifeline) makeSet(reasonRef reference.Holder) inspectsvc.RegisterRequestSet {

	rb, rootRec := p.recBuilder.MakeLineStart(&rms.RLifelineStart{
		Str: strconv.Itoa(int(p.seqNo.Add(1))),
	})
	rootRec.OverrideRecordType = rms.TypeRLifelineStartPolymorphID
	rootRec.OverrideReasonRef.Set(reasonRef)

	rMem := &rms.RLineMemoryInit{
		Polymorph: rms.TypeRLineMemoryInitPolymorphID,
		RootRef:   rootRec.AnticipatedRef,
		PrevRef:   rootRec.AnticipatedRef,
	}
	rMem.SetDigester(rb.RecordScheme.RecordDigester())
	rMem.SetPayload(rms.NewRawBytes(p.body))

	rq := rb.Add(rMem)

	rq = rb.Add(&rms.RLineActivate{
		RootRef: rootRec.AnticipatedRef,
		PrevRef: rq.AnticipatedRef,
	})

	return rb.MakeSet()
}

func (p *generatorNewLifeline) callRegister(recordSet inspectsvc.RegisterRequestSet) ([]cryptkit.Signature, error) {
	pn := p.recBuilder.RefTemplate.LocalHeader().Pulse()

	setSize := 0
	for _, r := range recordSet.Requests {
		setSize += r.ProtoSize()
		rp := r.GetRecordPayloads()
		setSize += rp.ProtoSize()
	}

	p.totalBytes.Add(uint64(setSize))

	ch := make(chan smachine.TerminationData, 1)
	err := p.conv.AddInputExt(pn,
		recordSet,
		smachine.CreateDefaultValues{
			Context: context.Background(),
			TerminationHandler: func(data smachine.TerminationData) {
				ch <- data
				close(ch)
			},
		})
	if err != nil {
		panic(err)
	}
	data := <-ch
	if data.Result == nil {
		return nil, data.Error
	}

	return data.Result.([]cryptkit.Signature), data.Error
}

// func (p *generatorNewLifeline) callVerify(recordSet inspectsvc.RegisterRequestSet) ([]cryptkit.Signature, error) {
// 	pn := p.recBuilder.RefTemplate.LocalHeader().Pulse()
//
// 	setSize := 0
// 	for _, r := range recordSet.Requests {
// 		setSize += r.ProtoSize()
// 		rp := r.GetRecordPayloads()
// 		setSize += rp.ProtoSize()
// 	}
//
// 	p.totalBytes.Add(uint64(setSize))
//
// 	ch := make(chan smachine.TerminationData, 1)
// 	err := p.conv.AddInputExt(pn,
// 		recordSet,
// 		smachine.CreateDefaultValues{
// 			Context: context.Background(),
// 			TerminationHandler: func(data smachine.TerminationData) {
// 				ch <- data
// 				close(ch)
// 			},
// 		})
// 	if err != nil {
// 		panic(err)
// 	}
// 	data := <-ch
// 	if data.Result == nil {
// 		return nil, data.Error
// 	}
//
// 	return data.Result.([]cryptkit.Signature), data.Error
// }

func (p *generatorNewLifeline) registerNewLine(reasonRef reference.Holder) ([]cryptkit.Signature, error) {
	recordSet := p.makeSet(reasonRef)
	return p.callRegister(recordSet)
}
