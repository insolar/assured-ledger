// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsec

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PlashSummaryWriter struct {
	recordToFilament []ledger.Ordinal
	filamentHeads    []FilamentHead
	merkleLog        []cryptkit.Digest // TODO

	summary          rms.ControlCatalogSummary
	receptacles      [4]bundle.PayloadReceptacle
	dirIndex         ledger.DirectoryIndex
}

func (p *PlashSummaryWriter) ReadCatalog(dirtyReader bundle.DirtyReader, dropCount int) error {
	directoryPages := dirtyReader.GetDirectoryEntries(ledger.DefaultEntrySection)

	lastPage := len(directoryPages) - 1
	pageSize := cap(directoryPages[0])
	totalCount := lastPage * pageSize + len(directoryPages[lastPage])

	p.recordToFilament = make([]ledger.Ordinal, totalCount)
	p.filamentHeads = make([]FilamentHead, 0, 1 + totalCount >> 4)
	filMap := make(map[ledger.Ordinal]int, cap(p.filamentHeads))

	for pageNo, page := range directoryPages {
		pageStart := pageNo * pageSize

		for i := range page {
			entry := page[i]
			entryOrd := ledger.Ordinal(pageStart + i)

			switch {
			case !entry.IsZero():
			case entryOrd == 0: // zero ordinal is unused
				continue
			default:
				// TODO set corruption marker?
				return throw.IllegalState()
			}

			head := entry.Fil.Link
			flags := entry.Fil.Flags

			if flags & ledger.FilamentLocalStart != 0 {
				p.recordToFilament[entryOrd] = entryOrd

				filMap[entryOrd] = len(p.filamentHeads)
				p.filamentHeads = append(p.filamentHeads, FilamentHead{
					Head:         entryOrd,
					FilamentInfo: entry.Fil,
				})
				if head == entryOrd {
					// head to itself
					continue
				}
			}

			var fil *FilamentHead
			switch filIdx, ok := filMap[head]; {
			case !ok:
				return throw.Impossible()
			default:
				fil = &p.filamentHeads[filIdx]
			}

			switch {
			case fil.Link >= entryOrd:
				return throw.Impossible()
			case fil.JetID != entry.Fil.JetID:
				return throw.Impossible()
			case flags & ledger.FilamentReopen != 0:
				fil.Flags &^= ledger.FilamentClosed
			case fil.Flags & ledger.FilamentClosed != 0:
				return throw.Impossible()
			}
			fil.Link = entryOrd
			p.recordToFilament[entryOrd] = head
		}
	}

	return nil
}

// PrepareWrite implementation should be much lighter, but there is only one summary written, so it doesn't matter much
func (p *PlashSummaryWriter) PrepareWrite(snapshot bundle.Snapshot) error {
	cp, err := snapshot.GetPayloadSection(ledger.ControlSection)
	if err != nil {
		return err
	}

	{
		sz := ordinalList(p.recordToFilament).Size()
		p.summary.RecToFilamentSize = uint32(sz)
		p.receptacles[0], p.summary.RecToFilamentLoc, err = cp.AllocatePayloadStorage(sz, ledger.SameAsBodyExtensionID)
		if err != nil {
			return err
		}
	}

	{
		sz := filamentHeads(p.filamentHeads).Size()
		p.summary.FilamentHeadsSize = uint32(sz)
		p.receptacles[1], p.summary.FilamentHeadsLoc, err = cp.AllocatePayloadStorage(sz, ledger.SameAsBodyExtensionID)
		if err != nil {
			return err
		}
	}

	// TODO merkleLog

	cd, err := snapshot.GetDirectorySection(ledger.ControlSection)
	if err != nil {
		return err
	}

	p.dirIndex = cd.GetNextDirectoryIndex()
	de := bundle.DirectoryEntry{ Key: reference.Global{}, } // TODO key
	p.receptacles[3], de.Loc, err = cd.AllocateEntryStorage(p.summary.ProtoSize())
	if err != nil {
		return err
	}

	return cd.AppendDirectoryEntry(p.dirIndex, de)
}

func (p *PlashSummaryWriter) ApplyWrite() ([]ledger.DirectoryIndex, error) {

	if r := p.receptacles[0]; r != nil {
		if _, err := r.WriteTo(ordinalList(p.recordToFilament)); err != nil {
			return nil, err
		}
	}
	if r := p.receptacles[1]; r != nil {
		if _, err := r.WriteTo(filamentHeads(p.filamentHeads)); err != nil {
			return nil, err
		}
	}
	// if r := p.receptacles[2]; r != nil {
	// 	// TODO merkleLog
	// }

	if err := p.receptacles[3].ApplyMarshalTo(&p.summary); err != nil {
		return nil, err
	}
	return []ledger.DirectoryIndex{p.dirIndex}, nil
}

func (p *PlashSummaryWriter) ApplyRollback() {}
