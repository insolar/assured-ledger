// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ctlsec

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newCtlEntryWriter(ref reference.Global, entry bundle.MarshalerTo) bundle.Writeable {
	if entry == nil {
		panic(throw.IllegalValue())
	}
	return &ctlEntryWriter{ ref: ref, entry: entry}
}

func newCtlEntryWriterExt(ref reference.Global, entryFn func (bundle.PayloadSection) (bundle.MarshalerTo, error)) bundle.Writeable {
	if entryFn == nil {
		panic(throw.IllegalValue())
	}
	return &ctlEntryWriter{ ref: ref, prepareFn: entryFn}
}

type ctlEntryWriter struct {
	ref              reference.Global
	prepareFn        func (bundle.PayloadSection) (bundle.MarshalerTo, error)
	entry            bundle.MarshalerTo
	dirIndex         ledger.DirectoryIndex
	receptacle       bundle.PayloadReceptacle
}

func (p *ctlEntryWriter) PrepareWrite(snapshot bundle.Snapshot) error {
	switch {
	case p.prepareFn != nil:
		cp, err := snapshot.GetPayloadSection(ledger.ControlSection)
		if err != nil {
			return err
		}
		p.entry, err = p.prepareFn(cp)
		if err != nil {
			return err
		}
	case p.entry == nil:
		return throw.IllegalState()
	}

	cd, err := snapshot.GetDirectorySection(ledger.ControlSection)
	if err != nil {
		return err
	}

	p.dirIndex = cd.GetNextDirectoryIndex()

	de := bundle.DirectoryEntry{ Key: p.ref }
	p.receptacle, de.Loc, err = cd.AllocateEntryStorage(p.entry.ProtoSize())
	if err != nil {
		return err
	}

	return cd.AppendDirectoryEntry(p.dirIndex, de)
}

func (p *ctlEntryWriter) ApplyWrite() ([]ledger.DirectoryIndex, error) {
	if err := p.receptacle.ApplyMarshalTo(p.entry); err != nil {
		return nil, err
	}
	return []ledger.DirectoryIndex{p.dirIndex}, nil
}

func (p *ctlEntryWriter) ApplyRollback() {}
