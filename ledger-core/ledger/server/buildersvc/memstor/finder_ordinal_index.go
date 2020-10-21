// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/ctlsection"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ readbundle.DirectoryIndexFinder = ordinalMapperFinder{}

type ordinalMapperFinder struct {
	name      string
	sectionID ledger.SectionID
	list 	  ctlsection.OrdinalListMapper
}

var ErrIndexUnknownOrdinal = throw.E("unknown ordinal by index")
var ErrIndexSectionMismatch = throw.E("section mismatch by index")

func (v ordinalMapperFinder) LookupByIndex(index ledger.DirectoryIndex) (_ ledger.DirectoryIndex, err error) {
	switch {
	case index == 0:
		return 0, nil
	case index.SectionID() != v.sectionID:
		err = ErrIndexSectionMismatch
	default:
		switch ord, ok := v.list.TryGet(index.Ordinal()); {
		case !ok:
			err = ErrIndexUnknownOrdinal
		case ord == 0:
			return 0, nil
		default:
			return ledger.NewDirectoryIndex(v.sectionID, ord), nil
		}
	}

	return 0, throw.WithDetails(err, struct {
		Name    string
		Section ledger.SectionID
		Lookup  ledger.DirectoryIndex
	}{ v.name, v.sectionID, index})
}

