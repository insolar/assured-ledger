// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsec

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type WriteAssistant struct {
	w bundle.Writer
	// signature hasher etc
}

func (p *WriteAssistant) Init(writer bundle.Writer, localDropCount int) {

}

func (p *WriteAssistant) WritePlashStart(pd pulse.Data, population census.OnlinePopulation) error {
	panic(throw.NotImplemented())
}

func (p *WriteAssistant) WriteSectionSummary(reader bundle.DirtyReader, section ledger.SectionID) error {
	sw := &SectionSummaryWriter{}
	if err := sw.ReadCatalog(reader, ledger.DefaultEntrySection); err != nil {
		return err
	}
	return p.w.WriteBundle(sw, nil)
}

func (p *WriteAssistant) WriteLineSummary(id jet.DropID, recap lineage.FilamentSummary, report rms.RStateReport) error {
	panic(throw.NotImplemented())
}

func (p *WriteAssistant) WriteFilamentSummary(id jet.DropID, fil lineage.FilamentSummary) error {
	panic(throw.NotImplemented())
}

func (p *WriteAssistant) WriteDropSummary(id jet.DropID) (catalog.DropReport, error) {
	panic(throw.NotImplemented())
}

func (p *WriteAssistant) WritePlashSummary() error {
	panic(throw.NotImplemented())
}

func (p *WriteAssistant) HasAllDropReports() bool {
	panic(throw.NotImplemented())
}
