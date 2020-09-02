// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/merkler"
)

func TestDropAssistAppend(t *testing.T) {
	local := gen.UniqueLocalRef()
	started := sync.WaitGroup{}
	completed := sync.WaitGroup{}
	hashed := sync.WaitGroup{}

	da := prepareDropAssistFoAppend(t, local, &started, &completed, &hashed)

	const bundleCount = 3
	started.Add(bundleCount)
	completed.Add(1)
	hashed.Add(bundleCount)

	pair := cryptkit.NewPairDigesterMock(t)
	pair.DigestPairMock.Set(func(digest0 longbits.FoldableReader, _ longbits.FoldableReader) cryptkit.Digest {
		return cryptkit.NewDigest(digest0, "")
	})

	pa := &plashAssistant{
		merkle: merkler.NewForkingCalculator(pair, cryptkit.Digest{}),
	}
	pa.status.Store(plashStarted)

	fts := make([]*Future, bundleCount)

	ref := reference.NewSelf(local)
	for i := range fts {
		fts[i] = NewFuture("test")
		err := da.append(pa, fts[i], newUpdateBundle(ref, byte(i+1)))
		require.NoError(t, err)
	}

	// all appends were prepared in parallel
	// but are blocked on completion
	started.Wait()

	// no futures are set yet
	for i := range fts {
		ok, err := fts[i].GetFutureResult()
		require.False(t, ok)
		require.NoError(t, err)
	}

	// allow all bundles to commit
	completed.Done()

	// and wait for them to be added to merkle tree
	// ordering is checked by merkle tree mock of dropAssist
	hashed.Wait()

	// then all futures should soon be ready
	for i := range fts {
		for j := 10; j > 0; j++ {
			ok, err := fts[i].GetFutureResult()
			if ok {
				require.NoError(t, err)
				break
			}
			time.Sleep(time.Duration(10-j)*100*time.Millisecond)
		}
	}
}

func TestDropAssistAppendWithPulseChange(t *testing.T) {
	local := gen.UniqueLocalRef()
	started := sync.WaitGroup{}
	completed := sync.WaitGroup{}
	hashed := sync.WaitGroup{}
	da := prepareDropAssistFoAppend(t, local, &started, &completed, &hashed)

	const bundleCount = 3
	started.Add(bundleCount)
	completed.Add(1)
	// hashed.Add(0)

	pair := cryptkit.NewPairDigesterMock(t)
	pair.DigestPairMock.Set(func(digest0 longbits.FoldableReader, _ longbits.FoldableReader) cryptkit.Digest {
		return cryptkit.NewDigest(digest0, "")
	})

	pa := &plashAssistant{
		// use of stub enables merkle tree to be empty
		// as we will rollback all operations
		merkle: merkler.NewForkingCalculator(pair, cryptkit.NewDigest(longbits.WrapStr("stub"), "testMerkle")),
	}
	pa.status.Store(plashStarted)

	fts := make([]*Future, bundleCount)

	ref := reference.NewSelf(local)
	for i := range fts {
		fts[i] = NewFuture("test")
		err := da.append(pa, fts[i], newUpdateBundle(ref, byte(i+1)))
		require.NoError(t, err)
	}

	// all appends were prepared in parallel
	// but are blocked on completion
	started.Wait()

	// no futures are set yet
	for i := range fts {
		ok, err := fts[i].GetFutureResult()
		require.False(t, ok)
		require.NoError(t, err)
	}

	stateHash := make(chan struct{})
	pa.PreparePulseChange(func(conveyor.PreparedState) {
		close(stateHash)
	})
	// we don't wait for pending bundles, as state hash is calculated before all pending bundles are released
	<- stateHash
	// nothing was added
	require.Equal(t, 0, pa.merkle.Count())

	// bundles are release, but they are blocked because of PreparePulseChange state
	completed.Done()

	// no one is ready
	for i := range fts {
		ok, err := fts[i].GetFutureResult()
		require.False(t, ok)
		require.NoError(t, err)
	}

	// this preserves merkle tree by forcing rollback for all pending bundles
	pa.CommitPulseChange()

	// all futures are set to an error
	for i := range fts {
		for j := 10; j > 0; j++ {
			ok, err := fts[i].GetFutureResult()
			if ok {
				require.Error(t, err)
				require.Equal(t, "plash closed", err.Error())
				break
			}
			time.Sleep(time.Duration(10-j)*100*time.Millisecond)
		}
	}

	// rollback was correct - nothing was added
	require.Equal(t, 0, pa.merkle.Count())
}

func TestDropAssistAppendWithPulseCancel(t *testing.T) {
	local := gen.UniqueLocalRef()
	started := sync.WaitGroup{}
	completed := sync.WaitGroup{}
	hashed := sync.WaitGroup{}
	da := prepareDropAssistFoAppend(t, local, &started, &completed, &hashed)

	const bundleCount = 3
	started.Add(bundleCount)
	completed.Add(1)
	hashed.Add(bundleCount)

	pair := cryptkit.NewPairDigesterMock(t)
	pair.DigestPairMock.Set(func(digest0 longbits.FoldableReader, _ longbits.FoldableReader) cryptkit.Digest {
		return cryptkit.NewDigest(digest0, "")
	})

	pa := &plashAssistant{
		// use of stub enables merkle tree to be empty
		merkle: merkler.NewForkingCalculator(pair, cryptkit.NewDigest(longbits.WrapStr("stub"), "testMerkle")),
	}
	pa.status.Store(plashStarted)

	fts := make([]*Future, bundleCount)

	ref := reference.NewSelf(local)
	for i := range fts {
		fts[i] = NewFuture("test")
		err := da.append(pa, fts[i], newUpdateBundle(ref, byte(i+1)))
		require.NoError(t, err)
	}

	// all appends were prepared in parallel
	// but are blocked on completion
	started.Wait()

	// no futures are set yet
	for i := range fts {
		ok, err := fts[i].GetFutureResult()
		require.False(t, ok)
		require.NoError(t, err)
	}

	stateHash := make(chan struct{})
	pa.PreparePulseChange(func(conveyor.PreparedState) {
		close(stateHash)
	})
	// we don't wait for pending bundles, as state hash is calculated before all pending bundles are released
	<- stateHash
	// nothing was added
	require.Equal(t, 0, pa.merkle.Count())

	// bundles are release, but they are blocked because of PreparePulseChange state
	completed.Done()

	// no one is ready
	for i := range fts {
		ok, err := fts[i].GetFutureResult()
		require.False(t, ok)
		require.NoError(t, err)
	}

	// here pulse change is cancelled, hence we have to proceed as normal
	// nothing has to be rolled back, bundles are released
	pa.CancelPulseChange()

	// and wait for them to be added to merkle tree
	// ordering is checked by merkle tree mock of dropAssist
	hashed.Wait()

	// then all futures should soon be ready
	for i := range fts {
		for j := 10; j > 0; j++ {
			ok, err := fts[i].GetFutureResult()
			if ok {
				require.NoError(t, err)
				break
			}
			time.Sleep(time.Duration(10-j)*100*time.Millisecond)
		}
	}
}

func prepareDropAssistFoAppend(t *testing.T, local reference.Local, started, completed, hashed *sync.WaitGroup) *dropAssistant {
	rcp := bundle.NewPayloadReceptacleMock(t)
	rcp.ApplyMarshalToMock.Return(nil)

	ps := bundle.NewPayloadSectionMock(t)
	payloadLoc := ledger.NewLocator(ledger.DefaultEntrySection, 1, 0)
	ps.AllocatePayloadStorageMock.Return(rcp, payloadLoc, nil)

	ds := bundle.NewDirectorySectionMock(t)
	dirIndex := ledger.NewDirectoryIndex(ledger.DefaultEntrySection, 1)
	ds.GetNextDirectoryIndexMock.Return(dirIndex)
	entryLock := ledger.NewLocator(ledger.DefaultEntrySection, 2, 0)
	ds.AllocateEntryStorageMock.Return(rcp, entryLock, nil)
	ds.AppendDirectoryEntryMock.Expect(dirIndex, reference.NewSelf(local), entryLock).Return(nil)

	snap := bundle.NewSnapshotMock(t)
	snap.GetDirectorySectionMock.Return(ds, nil)
	snap.GetPayloadSectionMock.Return(ps, nil)
	snap.PreparedMock.Return(nil)
	snap.CompletedMock.Set(func() error {
		started.Done()
		completed.Wait()
		return nil
	})
	snap.CommitMock.Return(nil)

	sw := bundle.NewSnapshotWriterMock(t)
	sw.TakeSnapshotMock.Return(snap, nil)

	lastId := 0
	merkle := cryptkit.NewForkingDigesterMock(t)
	merkle.AddNextMock.Set(func(digest longbits.FoldableReader) {
		require.Equal(t, 1, digest.FixedByteSize())
		id := int(digest.FoldToUint64())
		require.Equal(t, lastId + 1, id)
		lastId++
		hashed.Done()
	})

	return &dropAssistant{
		nodeID: 1,
		dropID: jet.ID(0).AsDrop(local.GetPulseNumber()),
		writer: bundle.NewWriter(sw),
		merkle: merkle,
	}
}

func newUpdateBundle(ref reference.Holder, id byte) lineage.UpdateBundle {
	rec := lineage.NewRegRecord(catalog.Excerpt{}, &rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(ref),
	})
	rec.RegistrarSignature = cryptkit.NewSignedDigest(
		cryptkit.NewDigest(longbits.WrapBytes([]byte{id}), "testDigestMethod"),
		cryptkit.NewSignature(longbits.WrapStr("signature"), "testSignMethod"))
	return lineage.NewUpdateBundleForTestOnly([]lineage.Record{rec})
}
