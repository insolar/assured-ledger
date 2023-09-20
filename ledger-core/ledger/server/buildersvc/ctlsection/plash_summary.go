package ctlsection

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SectionSummaryWriter struct {
	section          ledger.SectionID

	recordToFilament []ledger.Ordinal
	nextFilRecord    []ledger.Ordinal
	filamentHeads    []FilamentHead
	// merkleLog        []cryptkit.Digest // TODO

	summary          rms.RCtlSectionSummary
	dirIndex         ledger.DirectoryIndex

	receptRecToFil  bundle.PayloadReceptacle
	receptRecToPrev bundle.PayloadReceptacle
	receptFilToJet  bundle.PayloadReceptacle
	receptSummary   bundle.PayloadReceptacle
}

func (p *SectionSummaryWriter) ReadCatalog(dirtyReader bundle.DirtyReader, section ledger.SectionID) error {
	if section == ledger.ControlSection {
		panic(throw.IllegalValue())
	}
	p.section = section

	directoryPages := dirtyReader.GetDirectoryEntries(section)

	lastPage := len(directoryPages) - 1
	pageSize := cap(directoryPages[0])
	totalCount := lastPage * pageSize + len(directoryPages[lastPage])

	p.recordToFilament = make([]ledger.Ordinal, totalCount)
	p.nextFilRecord = make([]ledger.Ordinal, totalCount)
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
					First: entryOrd,
					Last:  entry.Fil.Link,
					JetID: entry.Fil.JetID,
					Flags: entry.Fil.Flags,
				})
				if head == entryOrd {
					// head to itself doesn't need to update head
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
			case fil.Last >= entryOrd:
				return throw.Impossible()
			case fil.JetID != entry.Fil.JetID:
				return throw.Impossible()
			case flags & ledger.FilamentReopen != 0:
				fil.Flags &^= ledger.FilamentClose
			case fil.Flags & ledger.FilamentClose != 0:
				return throw.Impossible()
			}
			if fil.Last > 0 {
				p.nextFilRecord[fil.Last] = entryOrd
			}
			fil.Last = entryOrd
			p.recordToFilament[entryOrd] = head
		}
	}

	return nil
}

// PrepareWrite implementation should be much lighter, but there is only one summary written, so it doesn't matter much
func (p *SectionSummaryWriter) PrepareWrite(snapshot bundle.Snapshot) error {
	if p.section == ledger.ControlSection {
		panic(throw.IllegalState())
	}

	cp, err := snapshot.GetPayloadSection(ledger.ControlSection)
	if err != nil {
		return err
	}

	{
		sz := ordinalList(p.recordToFilament).Size()
		p.summary.RecToFilSize = uint32(sz)
		p.receptRecToFil, p.summary.RecToFilLoc, err = cp.AllocatePayloadStorage(sz, ledger.SameAsBodyExtensionID)
		if err != nil {
			return err
		}
	}

	{
		sz := ordinalList(p.nextFilRecord).Size()
		p.summary.RecToNextSize = uint32(sz)
		p.receptRecToPrev, p.summary.RecToNextLoc, err = cp.AllocatePayloadStorage(sz, ledger.SameAsBodyExtensionID)
		if err != nil {
			return err
		}
	}

	{
		sz := filamentHeads(p.filamentHeads).Size()
		p.summary.FilToJetSize = uint32(sz)
		p.receptFilToJet, p.summary.FilToJetLoc, err = cp.AllocatePayloadStorage(sz, ledger.SameAsBodyExtensionID)
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

	de := bundle.DirectoryEntry{ Key: SectionCtlRecordRef(p.section, rms.TypeRCtlSectionSummaryPolymorphID) }
	p.receptSummary, de.Loc, err = cd.AllocateEntryStorage(p.summary.ProtoSize())
	if err != nil {
		return err
	}

	return cd.AppendDirectoryEntry(p.dirIndex, de)
}

func (p *SectionSummaryWriter) ApplyWrite() ([]ledger.DirectoryIndex, error) {

	if r := p.receptRecToFil; r != nil {
		if _, err := r.WriteTo(ordinalList(p.recordToFilament)); err != nil {
			return nil, err
		}
	}
	if r := p.receptRecToPrev; r != nil {
		if _, err := r.WriteTo(ordinalList(p.nextFilRecord)); err != nil {
			return nil, err
		}
	}
	if r := p.receptFilToJet; r != nil {
		if _, err := r.WriteTo(filamentHeads(p.filamentHeads)); err != nil {
			return nil, err
		}
	}
	// if r := p.receptacles[2]; r != nil {
	// 	// TODO merkleLog
	// }

	if err := p.receptSummary.ApplyMarshalTo(&p.summary); err != nil {
		return nil, err
	}
	return []ledger.DirectoryIndex{p.dirIndex}, nil
}

func (p *SectionSummaryWriter) ApplyRollback() {}
