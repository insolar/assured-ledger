// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp/lmntestapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

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

	t.Run("idempotency + verify", func(t *testing.T) {
		for N := 10; N > 0; N-- {
			// two intersecting bundles per object
			fullSet := genNewLine.makeSet(reasonRef)
			firstSet := fullSet

			// check for non-existence
			sg, err := genNewLine.callVerify(fullSet)
			require.NoError(t, err)
			require.Len(t, sg, 0)

			// add first record only
			firstSet.Requests = firstSet.Requests[:1]
			sg, err = genNewLine.callRegister(firstSet)
			require.NoError(t, err)
			require.Len(t, sg, 1)
			sgCopy := sg[0]

			// check for existence of the first record
			sg, err = genNewLine.callVerify(fullSet)
			require.NoError(t, err)
			require.Len(t, sg, 1)
			require.True(t, sgCopy.Equals(sg[0]))

			// add all records together, first record must be deduplicated
			sg, err = genNewLine.callRegister(fullSet)
			require.NoError(t, err)
			require.Len(t, sg, 3)
			require.True(t, sgCopy.Equals(sg[0]))

			bundleSignatures = append(bundleSignatures, sg)

			sgCopyAll := sg

			// check for existence of all records
			sg, err = genNewLine.callVerify(fullSet)
			require.NoError(t, err)
			require.Len(t, sg, 3)

			for j := range sg {
				require.True(t, sgCopyAll[j].Equals(sg[j]), j)
			}
		}
	})

	// repeat the same sequence
	genNewLine.seqNo.Store(0)

	// all registrations must be ok as they will be deduplicated
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

	// repeat the same sequence
	genNewLine.seqNo.Store(0)

	t.Run("read", func(t *testing.T) {
		for N := 20; N > 0; N-- {
			recordSet := genNewLine.makeSet(reasonRef)

			// take the last and read to first
			resp, err := genNewLine.callRead(recordSet.Requests[2].AnticipatedRef.Get())
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Entries, 3)

			for i := range resp.Entries {
				expected := recordSet.Requests[2-i]
				actual := &resp.Entries[i]
				require.Equal(t, expected.AnticipatedRef.GetValue(), actual.RecordRef.GetValue(), i)
				require.NotEmpty(t, actual.RecordBinary)

				require.True(t, expected.AnyRecordLazy.TryGetLazy().EqualBytes(actual.RecordBinary))
			}
		}
	})
}

func TestAddRecordsThenChangePulse(t *testing.T) {
	server := lmntestapp.NewTestServer(t)
	defer server.Stop()

	var recBuilder lmntestapp.RecordBuilder

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		recBuilder = lmntestapp.NewRecordBuilderFromDependencies(params.AppInject)
	})

	server.Start()

	var readSvc readersvc.Service
	{
		var readAdapter readersvc.Adapter
		server.Injector().MustInject(&readAdapter)
		readSvc = readersvc.GetServiceForTestOnly(readAdapter)
	}

	server.RunGenesis()
	server.IncrementPulse()

	recBuilder.RefTemplate = reference.NewSelfRefTemplate(server.LastPulseNumber(), reference.SelfScopeLifeline)

	genNewLine := generatorNewLifeline{
		recBuilder: recBuilder,
		conv:       server.App().Conveyor(),
		body:       make([]byte, 1<<10),
	}

	pn := server.LastPulseNumber()
	reasonRef := gen.UniqueGlobalRefWithPulse(pn)

	var bundleSignatures [][]cryptkit.Signature

	for N := 10; N > 0; N-- {
		// one bundle per object
		sg, err := genNewLine.registerNewLine(reasonRef)
		require.NoError(t, err)
		require.Len(t, sg, 3)
		bundleSignatures = append(bundleSignatures, sg)
	}

	server.IncrementPulse()

	time.Sleep(2*time.Second)

	server.IncrementPulse()

	require.Eventually(t, func() bool {
		return readSvc.FindCabinet(pn) != nil
	}, 2*time.Second, 10*time.Millisecond)
}

func BenchmarkWriteNew(b *testing.B) {
	b.Run("1k", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<10, false)
	})

	b.Run("1k-par", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<10, true)
	})

	b.Run("16k", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<14, false)
	})

	b.Run("16k-par", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<14, true)
	})

	b.Run("128k", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<17, false)
	})

	b.Run("128k-par", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<17, true)
	})

	b.Run("1M", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<20, false)
	})

	b.Run("1M-par", func(b *testing.B) {
		benchmarkWriteRead(b, 1<<20, true)
	})
}

func benchmarkWriteRead(b *testing.B, bodySize int, parallel bool) {
	server := lmntestapp.NewTestServer(b)
	defer server.Stop()

	var recBuilder lmntestapp.RecordBuilder

	server.SetImposer(func(params *insconveyor.ImposedParams) {
		recBuilder = lmntestapp.NewRecordBuilderFromDependencies(params.AppInject)
	})

	server.Start()
	server.RunGenesis()
	server.IncrementPulse()

	reasonRef := gen.UniqueGlobalRefWithPulse(server.LastPulseNumber())

	maxSeqNo := uint32(0)

	b.Run("write", func(b *testing.B) {
		recBuilder.RefTemplate = reference.NewSelfRefTemplate(server.LastPulseNumber(), reference.SelfScopeLifeline)
		genNewLine := generatorNewLifeline{
			recBuilder: recBuilder,
			conv:       server.App().Conveyor(),
			body:       make([]byte, bodySize),
		}

		b.ResetTimer()
		b.ReportAllocs()

		if parallel {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					recordSet := genNewLine.makeSet(reasonRef)
					genNewLine.sumUpRegister(recordSet)
					_, _ = genNewLine.callRegister(recordSet)
				}
			})
		} else {
			for i := b.N; i > 0; i-- {
				recordSet := genNewLine.makeSet(reasonRef)
				genNewLine.sumUpRegister(recordSet)
				_, _ = genNewLine.callRegister(recordSet)
			}
		}

		maxSeqNo = genNewLine.seqNo.Load()
		b.SetBytes(int64(genNewLine.totalBytes.Load()) / int64(maxSeqNo))
	})

	if maxSeqNo == 0 {
		panic(throw.IllegalState())
	}

	maxSeqNoPtr := &maxSeqNo // mitigate Go's bug
	b.Run("read", func(b *testing.B) {
		maxReadSeqNo := *maxSeqNoPtr
		if maxReadSeqNo == 0 {
			panic(throw.IllegalState())
		}

		recBuilder.RefTemplate = reference.NewSelfRefTemplate(server.LastPulseNumber(), reference.SelfScopeLifeline)
		genNewLine := generatorNewLifeline{
			recBuilder: recBuilder,
			conv:       server.App().Conveyor(),
			body:       make([]byte, bodySize),
		}

		refs := make([]reference.Holder, 0, maxReadSeqNo)

		for ; maxReadSeqNo > 0; maxReadSeqNo-- {
			recordSet := genNewLine.makeSet(reasonRef)
			refs = append(refs, recordSet.Requests[2].AnticipatedRef.Get())
		}

		b.ResetTimer()
		b.ReportAllocs()

		iterCount := atomickit.Uint32{}
		if parallel {
			b.RunParallel(func(pb *testing.PB) {
				idx := rand.Intn(len(refs))
				for pb.Next() {
					resp, _ := genNewLine.callRead(refs[idx])
					genNewLine.sumUpRead(resp)
					idx++
					idx %= len(refs)
					iterCount.Add(1)
				}
			})
		} else {
			idx := rand.Intn(len(refs))
			for i := b.N; i > 0; i-- {
				resp, _ := genNewLine.callRead(refs[idx])
				genNewLine.sumUpRead(resp)
				idx++
				idx %= len(refs)
				iterCount.Add(1)
			}
		}

		b.SetBytes(int64(genNewLine.totalBytes.Load()) / int64(iterCount.Load()))
	})
}
