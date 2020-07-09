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
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/cabinet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type dropAssistant struct {
	// set at construction
	nodeID node.ShortNodeID
	dropID jet.DropID
	writer cabinet.BundleWriter

	mutex   sync.Mutex // LOCK! Is used under plashAssistant.commit lock
	merkle  cryptkit.ForkingDigester
}

func (p *dropAssistant) append(pa *plashAssistant, future AppendFuture, bundle lineage.ResolvedBundle) (err error) {
	writeBundle := make([]cabinet.WriteBundleEntry, 0, bundle.Count())
	digests := make([]cryptkit.Digest, 0, bundle.Count())

	bundle.Enum(func(record lineage.Record, dust lineage.DustMode) bool {
		br := record.AsBasicRecord()
		recPayloads := br.GetRecordPayloads()
		payloadCount := recPayloads.Count()

		bundleEntry := cabinet.WriteBundleEntry{
			Directory: ledger.DefaultEntrySection, // todo depends on record policy
			EntryKey:  record.GetRecordRef(),
			Payloads:  make([]cabinet.SectionPayload, 1+payloadCount),
		}

		if dust == lineage.DustRecord {
			bundleEntry.Directory = ledger.DefaultDustSection
		}

		bundleEntry.Payloads[0].Section = bundleEntry.Directory
		if mt, ok := br.(cabinet.MarshalerTo); ok {
			bundleEntry.Payloads[0].Payload = mt
		} else {
			err = throw.E("incompatible record")
			return true // stop now
		}

		if payloadCount > 0 {
			if dust >= lineage.DustPayload {
				bundleEntry.Payloads[1].Section = ledger.DefaultDustSection
			} else {
				bundleEntry.Payloads[1].Section = ledger.DefaultDataSection
			}
			bundleEntry.Payloads[1].Payload = recPayloads.GetPayloadOrExtension(0)

			for i := 2; i <= payloadCount; i++ {
				secID := bundleEntry.Directory
				extID := ledger.ExtensionID(recPayloads.GetExtensionID(i - 1))
				if extID != ledger.SameAsBodyExtensionID {
					secID = p.extensionToSection(extID, secID)
				}
				bundleEntry.Payloads[i].Extension = extID
				bundleEntry.Payloads[i].Section = secID
				bundleEntry.Payloads[i].Payload = recPayloads.GetPayloadOrExtension(i - 1)
			}
		}

		bundleEntry.EntryFn = func(index ledger.DirectoryIndex, payloadLoc []ledger.StorageLocator) cabinet.MarshalerTo {
			entry := createCatalogEntry(index, payloadLoc, bundleEntry.Payloads, &record)
			return entry
		}

		digests = append(digests, record.RegistrarSignature.GetDigest())
		writeBundle = append(writeBundle, bundleEntry)
		return false
	})
	if err != nil {
		return
	}

	return p.writer.WriteBundle(writeBundle, func(indices []ledger.DirectoryIndex, err error) bool {
		// this closure is called later, after the bundle is completely written
		// this closure can be called twice, when rollback was requested, but has failed - the the 2nd call will be with an error
		if err == nil {
			err = pa.commitDropUpdate(func() error {
				// EXTREME LOCK WARNING!
				// This section is under locks of: (1) BundleWriter, (2) plashAssistant, and acquires (3) dropAssistant.
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
// This method is under locks of: (1) BundleWriter, (2) plashAssistant, (3) dropAssistant.
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
// This method is under locks of: (1) BundleWriter, (2) plashAssistant, (3) dropAssistant.
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

func createCatalogEntry(idx ledger.DirectoryIndex, loc []ledger.StorageLocator, payloads []cabinet.SectionPayload, rec *lineage.Record) *catalog.Entry {
	entry := &catalog.Entry{
		RecordType:         rec.Excerpt.RecordType,
		BodyLoc:            loc[0],
		RecordBodyHash:     rec.Excerpt.RecordBodyHash,
		Ordinal: 			idx.Ordinal(),
		PrevRef:			rec.Excerpt.PrevRef,
		RootRef:			rec.Excerpt.RootRef,
		ReasonRef:			rec.Excerpt.ReasonRef,
		RedirectRef:		rec.Excerpt.RedirectRef,
		RejoinRef:			rec.Excerpt.RejoinRef,
		RecapRef: 			rms.NewReference(rec.RecapRef),

		ProducerSignature:  rec.Excerpt.ProducerSignature,
		ProducedBy:         rms.NewReference(rec.ProducedBy),

		RegistrarSignature: rms.NewRaw(rec.RegistrarSignature.GetSignature()).AsBinary(),
		RegisteredBy:       rms.NewReference(rec.RegisteredBy),
	}

	if n := len(loc); n > 1 {
		entry.PayloadLoc = loc[1]
		if n > 2 {
			entry.ExtensionLoc.Ext = make([]rms.ExtLocator, n - 2)
			for i := 2; i < n; i++ {
				entry.ExtensionLoc.Ext[i - 2] = rms.ExtLocator{
					ExtensionID: payloads[i].Extension,
					PayloadLoc:  loc[i],
				}
			}
		}
	}

	return entry
}

