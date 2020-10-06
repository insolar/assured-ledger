// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsection

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type WriteAssistant struct {
	w          bundle.Writer
	callbackFn func (error)
	// signature hasher etc
	expectedDropCount int

	mutex sync.Mutex
	state    atomickit.StartStopFlag
	start    ledger.Ordinal
	summary  ledger.Ordinal
	sections []ledger.Ordinal
	drops    []ledger.Ordinal
}

func (p *WriteAssistant) Init(writer bundle.Writer, localDropCount int, callbackFn func (error)) {
	switch {
	case writer == nil:
		panic(throw.IllegalValue())
	case callbackFn == nil:
		panic(throw.IllegalValue())
	case localDropCount < 0:
		panic(throw.IllegalValue())
	}

	p.callbackFn = callbackFn
	p.expectedDropCount = localDropCount
	p.w = writer
}

func (p *WriteAssistant) ensureOpen() {
	if !p.state.IsActive() {
		panic(throw.IllegalState())
	}
}

func (p *WriteAssistant) WritePlashStart(pd pulse.Data, population census.OnlinePopulation) (err error) {
	switch {
	case !pd.IsFromPulsar():
		panic(throw.IllegalValue())
	case pd.PulseNumber.AsEpoch() != pd.PulseEpoch:
		panic(throw.NotImplemented())
	}

	if !p.state.DoStart(func() {
		startEntry := &rms.RCtlPlashStart{
			Version:        SerializationVersion,
			// NodeRef:        rms.Reference{},
			// PulseData:      rms.Reference{},
			// PulseEpochData: rms.Reference{},
			// PopulationRef:  rms.Reference{},
			// Population:     rms.Binary{},
		}

		ew := newCtlEntryWriter(CtlSectionRef(ledger.ControlSection, false), startEntry)

		err = p.w.WriteBundle(ew, func(indices []ledger.DirectoryIndex, err error) bool {
			if err != nil {
				p.callbackFn(err)
				return false
			}

			p.mutex.Lock()
			defer p.mutex.Unlock()
			p.start = indices[0].Ordinal()

			return true
		})
	}) {
		panic(throw.IllegalState())
	}

	if err == nil && p.expectedDropCount == 0 {
		return p.WritePlashSummary()
	}
	return err
}

func (p *WriteAssistant) WriteSectionSummary(reader bundle.DirtyReader, section ledger.SectionID) error {
	p.ensureOpen()

	sw := &SectionSummaryWriter{}
	if err := sw.ReadCatalog(reader, section); err != nil {
		return err
	}
	return p.w.WriteBundle(sw, func(indices []ledger.DirectoryIndex, err error) bool {
		if err != nil {
			p.callbackFn(err)
			return false
		}

		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.sections = append(p.sections, indices[0].Ordinal())

		return true
	})
}

func (p *WriteAssistant) WriteLineSummary(id jet.DropID, recap lineage.FilamentSummary, report rms.RStateReport) error {
	p.ensureOpen()

	lineEntry := &rms.RCtlFilamentEntry{
		LastKnownPN:             id.CreatedAt(),
		LineRecap:               recap.Recap,
//		RecapProducerSignature:  rms.Binary{},
		LineReport:              report,
//		ReportProducerSignature: rms.Binary{},
	}

	ew := newCtlEntryWriter(reference.NewSelf(recap.Local), lineEntry)

	return p.w.WriteBundle(ew, func(_ []ledger.DirectoryIndex, err error) bool {
		if err != nil {
			p.callbackFn(err)
			return false
		}
		return true
	})
}

func (p *WriteAssistant) WriteFilamentSummary(id jet.DropID, lineBase reference.Local, fil lineage.FilamentSummary) error {
	p.ensureOpen()

	lineEntry := &rms.RCtlFilamentEntry{
		LastKnownPN:             id.CreatedAt(),
		LineRecap:               fil.Recap,
		//		RecapProducerSignature:  rms.Binary{},
	}

	ew := newCtlEntryWriter(reference.New(lineBase, fil.Local), lineEntry)

	return p.w.WriteBundle(ew, func(_ []ledger.DirectoryIndex, err error) bool {
		if err != nil {
			p.callbackFn(err)
			return false
		}
		return true
	})
}

func (p *WriteAssistant) WriteDropSummary(id jet.DropID, finalizeFn func ()) (catalog.DropReport, error) {
	p.ensureOpen()

	dropEntry := &rms.RCtlDropSummary{
		// DropReport:              rms.RCtlDropReport{},
		// ReportProducerSignature: rms.Binary{},
		// MerkleLogLoc:            0,
		// MerkleLogSize:           0,
		// MerkleLogCount:          0,
	}

	ew := newCtlEntryWriter(JetRef(id.ID()), dropEntry)
	dr := catalog.DropReport{ ReportRec: &dropEntry.DropReport }

	err := p.w.WriteBundle(ew, func(indices []ledger.DirectoryIndex, err error) bool {
		if err != nil {
			p.callbackFn(err)
			return false
		}

		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.drops = append(p.drops, indices[0].Ordinal())
		if len(p.drops) >= p.expectedDropCount && finalizeFn != nil {
			go finalizeFn()
		}

		return true
	})

	return dr, err
}

func (p *WriteAssistant) WritePlashSummary() (err error) {
	if !p.state.DoStop(func() {
		p.mutex.Lock()
		plashSummary := &rms.RCtlPlashSummary{
			// MerkleRoot:              rms.Binary{},
			// MerkleProducerSignature: rms.Binary{},
			SectionSummaryOrd: append([]ledger.Ordinal(nil), p.sections...),
			DropSummaryOrd:    append([]ledger.Ordinal(nil), p.drops...),
		}
		p.mutex.Unlock()

		ew := newCtlEntryWriter(CtlSectionRef(ledger.ControlSection, true), plashSummary)

		err = p.w.WriteBundle(ew, func(indices []ledger.DirectoryIndex, err error) bool {
			if err != nil {
				p.callbackFn(err)
				return false
			}

			p.mutex.Lock()
			defer p.mutex.Unlock()
			p.summary = indices[0].Ordinal()

			return true
		})
	}) {
		panic(throw.IllegalState())
	}

	return err
}
