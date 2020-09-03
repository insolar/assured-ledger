// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestMemorySnapshot(t *testing.T) {
	ms := NewMemoryStorageWriter(ledger.DefaultDustSection, directoryEntrySize*16)
	s, _ := ms.TakeSnapshot()

	es, err := s.GetDirectorySection(ledger.DefaultEntrySection)
	require.NoError(t, err)

	r, loc, err := es.AllocateEntryStorage(10)
	require.NoError(t, err)
	require.Equal(t, ledger.NewLocator(ledger.DefaultEntrySection, 1, 0), loc)
	require.NotNil(t, r)

	err = r.ApplyFixedReader(longbits.WrapStr("0123456789"))
	require.NoError(t, err)

	nextIdx := es.GetNextDirectoryIndex()
	require.Equal(t, ledger.NewDirectoryIndex(ledger.DefaultEntrySection, 1), nextIdx)
	require.Equal(t, nextIdx, es.GetNextDirectoryIndex())

	require.Panics(t, func() {
		_ = es.AppendDirectoryEntry(0, bundle.DirectoryEntry{ Key: gen.UniqueGlobalRef(), Loc: loc})
	})

	err = es.AppendDirectoryEntry(nextIdx, bundle.DirectoryEntry{Key: gen.UniqueGlobalRef(), Loc: loc})
	require.NoError(t, err)

	r0 := r.(byteReceptacle)
	r, loc, err = es.AllocateEntryStorage(16)
	require.NoError(t, err)
	require.Equal(t, ledger.NewLocator(ledger.DefaultEntrySection, 1, 11), loc)
	require.NotNil(t, r)

	err = r.ApplyFixedReader(longbits.WrapStr("0123456789"))
	require.Error(t, err)
	err = r.ApplyFixedReader(longbits.WrapStr("0123456789ABCDEF"))
	require.NoError(t, err)

	_ = append(r0, "overflow"...) // check that an allocated slice is protected from overflow

	// Directory entry is appended with varint size prefix
	require.Equal(t, "\x0a0123456789\x100123456789ABCDEF", string(ms.sections[ledger.DefaultEntrySection].chapters[0]))
}

func TestMemorySnapshotDirectoryPaging(t *testing.T) {
	ms := NewMemoryStorageWriter(ledger.DefaultDustSection, directoryEntrySize*16)
	s, _ := ms.TakeSnapshot()
	es, err := s.GetDirectorySection(ledger.DefaultEntrySection)
	require.NoError(t, err)

	loc := ledger.NewLocator(ledger.DefaultEntrySection, 1, 1)

	j := ledger.Ordinal(1)
	for i := cap(ms.sections[ledger.DefaultEntrySection].directory[0])*3; i > 0; i-- {
		nextIdx := es.GetNextDirectoryIndex()
		require.Equal(t, ledger.NewDirectoryIndex(ledger.DefaultEntrySection, j), nextIdx)
		require.Equal(t, nextIdx, es.GetNextDirectoryIndex())

		err = es.AppendDirectoryEntry(nextIdx, bundle.DirectoryEntry{Key: gen.UniqueGlobalRef(), Loc: loc})
		require.NoError(t, err)
		j++
		require.Equal(t, ledger.NewDirectoryIndex(ledger.DefaultEntrySection, j), es.GetNextDirectoryIndex())
	}

	require.True(t, len(ms.sections[ledger.DefaultEntrySection].directory) > 1)

	sm := s.(*memorySnapshot)
	require.Equal(t, int(1 + ledger.DefaultDustSection), len(sm.snapshot))
	ess := &sm.snapshot[ledger.DefaultEntrySection]
	require.Equal(t, &ms.sections[ledger.DefaultEntrySection], ess.section)
	require.Equal(t, ledger.ChapterID(0), ess.chapter)
	require.Equal(t, ledger.Ordinal(1), ess.dirIndex)

	s, _ = ms.TakeSnapshot()
	es, err = s.GetDirectorySection(ledger.DefaultEntrySection)
	require.NoError(t, err)

	nextIdx := es.GetNextDirectoryIndex()
	err = es.AppendDirectoryEntry(nextIdx, bundle.DirectoryEntry{Key: gen.UniqueGlobalRef(), Loc: loc})
	require.NoError(t, err)

	sm = s.(*memorySnapshot)
	require.Equal(t, int(1 + ledger.DefaultDustSection), len(sm.snapshot))
	ess = &sm.snapshot[ledger.DefaultEntrySection]
	require.Equal(t, &ms.sections[ledger.DefaultEntrySection], ess.section)
	require.Equal(t, ledger.ChapterID(0), ess.chapter)
	require.Equal(t, j, ess.dirIndex)

	require.Equal(t, ledger.NewDirectoryIndex(ledger.DefaultEntrySection, j + 1), es.GetNextDirectoryIndex())

	s.Rollback(false)

	require.Equal(t, ledger.NewDirectoryIndex(ledger.DefaultEntrySection, j), es.GetNextDirectoryIndex())
}

func TestMemorySnapshotPayloadPaging(t *testing.T) {
	pageSize := directoryEntrySize*16
	ms := NewMemoryStorageWriter(ledger.DefaultDustSection, pageSize)
	s, _ := ms.TakeSnapshot()
	es, err := s.GetPayloadSection(ledger.DefaultEntrySection)
	require.NoError(t, err)

	allocSize := pageSize / 4 - 1
	allocated := 0
	for i := 4; i > 0; i -- {
		r, loc, err := es.AllocatePayloadStorage(allocSize, 0)
		require.NoError(t, err)
		require.NotNil(t, r)
		require.Equal(t, ledger.NewLocator(ledger.DefaultEntrySection, 1, uint32(allocated)), loc)
		allocated += allocSize
	}

	r, loc, err := es.AllocatePayloadStorage(allocSize, 0)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, ledger.NewLocator(ledger.DefaultEntrySection, 2, 0), loc)

	sm := s.(*memorySnapshot)
	ess := &sm.snapshot[ledger.DefaultEntrySection]
	require.Equal(t, &ms.sections[ledger.DefaultEntrySection], ess.section)
	require.Equal(t, ledger.Ordinal(0), ess.dirIndex)
	require.Equal(t, ledger.ChapterID(1), ess.chapter)
	require.Equal(t, uint32(0), ess.lastOfs)

	s, _ = ms.TakeSnapshot()
	ess = &s.(*memorySnapshot).snapshot[ledger.DefaultEntrySection]
	require.Nil(t, ess.section)
	require.Equal(t, ledger.Ordinal(0), ess.dirIndex)
	require.Equal(t, ledger.ChapterID(0), ess.chapter)

	es, _ = s.GetPayloadSection(ledger.DefaultEntrySection)
	_, _, err = es.AllocatePayloadStorage(1, 0)
	require.NoError(t, err)

	r, loc, err = es.AllocatePayloadStorage(pageSize, 0)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, ledger.NewLocator(ledger.DefaultEntrySection, 3, 0), loc)

	r, loc, err = es.AllocatePayloadStorage(pageSize*3, 0)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, ledger.NewLocator(ledger.DefaultEntrySection, 4, 0), loc)

	chapters := ms.sections[ledger.DefaultEntrySection].chapters
	require.Equal(t, 4, len(chapters))
	require.Equal(t, 4*allocSize, len(chapters[0]))
	require.Equal(t, allocSize + 1, len(chapters[1]))
	require.Equal(t, pageSize, len(chapters[2]))
	require.Equal(t, 3*pageSize, len(chapters[3]))

	ess = &s.(*memorySnapshot).snapshot[ledger.DefaultEntrySection]
	require.Equal(t, &ms.sections[ledger.DefaultEntrySection], ess.section)
	require.Equal(t, ledger.Ordinal(0), ess.dirIndex)
	require.Equal(t, ledger.ChapterID(2), ess.chapter)
	require.Equal(t, uint32(allocSize), ess.lastOfs)

	s.Rollback(false)

	chapters = ms.sections[ledger.DefaultEntrySection].chapters
	require.Equal(t, 2, len(chapters))
	require.Equal(t, 4*allocSize, len(chapters[0]))
	require.Equal(t, allocSize, len(chapters[1]))
}

type benchBundle struct {
	data  []byte
	recep bundle.PayloadReceptacle
}

func (p *benchBundle) PrepareWrite(snapshot bundle.Snapshot) error {
	ps, err := snapshot.GetPayloadSection(ledger.DefaultEntrySection)
	if err == nil {
		p.recep, _, err = ps.AllocatePayloadStorage(len(p.data), 0)
	}
	return err
}

func (p *benchBundle) ApplyWrite() ([]ledger.DirectoryIndex, error) {
	err := p.recep.ApplyFixedReader(longbits.WrapBytes(p.data))
	return nil, err
}

func BenchmarkMemoryStorageWrite(b *testing.B) {
	ms := NewMemoryStorageWriter(ledger.DefaultEntrySection, 1<<16)
	mw := bundle.NewWriter(ms)
	src := make([]byte, 1<<12)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := b.N; i > 0; i-- {
		ch := make(chan struct{})
		_ = mw.WriteBundle(&benchBundle{data: src}, func(_ []ledger.DirectoryIndex, err error) bool {
			defer close(ch)
			if err != nil {
				panic(err)
			}
			return true
		})
		<- ch
	}
}

func BenchmarkMemoryStorageParallelWrite(b *testing.B) {
	ms := NewMemoryStorageWriter(ledger.DefaultEntrySection, 1<<16)
	mw := bundle.NewWriter(ms)
	src := make([]byte, 1<<12)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	b.SetParallelism(8)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch := make(chan struct{})
			_ = mw.WriteBundle(&benchBundle{data: src}, func(_ []ledger.DirectoryIndex, err error) bool {
				defer close(ch)
				if err != nil {
					panic(err)
				}
				return true
			})
			<- ch
		}
	})
}

func TestDirectoryEntrySize(t *testing.T) {
	require.EqualValues(t, directoryEntrySize, unsafe.Sizeof(bundle.DirectoryEntry{}))
}
