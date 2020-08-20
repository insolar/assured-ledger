// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type dropAssistant struct {
	// set at construction
	nodeID node.ShortNodeID
	dropID jet.DropID
	writer bundle.Writer

	mutex   sync.Mutex // LOCK: Is used under plashAssistant.commit lock
	merkle  cryptkit.ForkingDigester
}

func (p *dropAssistant) append(pa *plashAssistant, future AppendFuture, b lineage.UpdateBundle) (err error) {
	entries := make([]draftEntry, 0, b.Count())
	digests := make([]cryptkit.Digest, 0, b.Count())

	b.Enum(func(record lineage.Record, dust lineage.DustMode) bool {
		br := record.AsBasicRecord()
		recPayloads := br.GetRecordPayloads()
		payloadCount := recPayloads.Count()

		bundleEntry := draftEntry{
			directory: ledger.DefaultEntrySection, // todo depends on record policy
			entryKey:  record.GetRecordRef(),
			payloads:  make([]sectionPayload, 1+payloadCount),
			draft:     draftCatalogEntry(record),
		}

		if dust == lineage.DustRecord {
			bundleEntry.directory = ledger.DefaultDustSection
		}

		bundleEntry.payloads[0].section = bundleEntry.directory
		if mt, ok := br.(bundle.MarshalerTo); ok {
			bundleEntry.payloads[0].payload = mt
		} else {
			err = throw.E("incompatible record")
			return true // stop now
		}

		if payloadCount > 0 {
			if dust >= lineage.DustPayload {
				bundleEntry.payloads[1].section = ledger.DefaultDustSection
			} else {
				bundleEntry.payloads[1].section = ledger.DefaultDataSection
			}
			bundleEntry.payloads[1].payload = recPayloads.GetPayloadOrExtension(0)

			for i := 2; i <= payloadCount; i++ {
				secID := bundleEntry.directory
				extID := ledger.ExtensionID(recPayloads.GetExtensionID(i - 1))
				if extID != ledger.SameAsBodyExtensionID {
					secID = p.extensionToSection(extID, secID)
				}
				bundleEntry.payloads[i].extension = extID
				bundleEntry.payloads[i].section = secID
				bundleEntry.payloads[i].payload = recPayloads.GetPayloadOrExtension(i - 1)
			}
		}

		digests = append(digests, record.RegistrarSignature.GetDigest())
		entries = append(entries, bundleEntry)
		return false
	})
	if err != nil {
		return
	}

	writeBundle := &entryWriter{entries: entries}

	return p.writer.WriteBundle(writeBundle, func(indices []ledger.DirectoryIndex, err error) bool {
		// this closure is called later, after the bundle is completely written
		// this closure can be called twice, when rollback was requested, but has failed - the the 2nd call will be with an error
		if err == nil {
			err = pa.commitDropUpdate(func() error {
				// EXTREME LOCK WARNING!
				// This section is under locks of: (1) bundle.Writer, (2) plashAssistant, and acquires (3) dropAssistant.
				return p.bundleProcessedByWriter(pa, indices, digests)
			})
		}

		if err != nil {
			go future.TrySetFutureResult(nil, err)
		} else {
			go future.TrySetFutureResult(indices, nil)
		}
		return true
	})
}

// EXTREME LOCK WARNING!
// This method is under locks of: (1) bundle.Writer, (2) plashAssistant, (3) dropAssistant.
func (p *dropAssistant) bundleProcessedByWriter(pa *plashAssistant, indices []ledger.DirectoryIndex, digests []cryptkit.Digest) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	positions, err := pa._updateMerkle(p.dropID, indices, digests)
	if err != nil {
		return err
	}
	p._updateMerkle(positions, indices, digests)
	return nil
}

// EXTREME LOCK WARNING!
// This method is under locks of: (1) bundle.Writer, (2) plashAssistant, (3) dropAssistant.
func (p *dropAssistant) _updateMerkle(_ []ledger.Ordinal, indices []ledger.DirectoryIndex, digests []cryptkit.Digest) {
	if p.merkle == nil {
		// there is only one drop in plash, so there is no need for a secondary merkle
		return
	}

	for i, ord := range indices {
		if ord.SectionID() != ledger.DefaultEntrySection {
			continue
		}
		p.merkle.AddNext(digests[i])
		// TODO pre-calculate sparse merkle proof for this drop by plash
	}
}

func (p *dropAssistant) extensionToSection(_ ledger.ExtensionID, defSecID ledger.SectionID) ledger.SectionID {
	// TODO extension mapping
	return defSecID
}
