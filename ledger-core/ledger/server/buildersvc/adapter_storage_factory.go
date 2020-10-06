// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dropstorage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newStorageFactory(maxSection ledger.SectionID, pageSize int) storageFactory {
	switch {
	case maxSection < ledger.DefaultDataSection:
		panic(throw.IllegalValue())
	case maxSection > ledger.MaxSectionID:
		panic(throw.IllegalValue())
	case pageSize < dropstorage.MinStoragePageSize:
		panic(throw.IllegalValue())
	}
	return storageFactory{maxSection, pageSize}
}

type storageFactory struct {
	maxSection ledger.SectionID
	pageSize int
}

func (v storageFactory) CreateSnapshotWriter(pulse.Number) bundle.SnapshotWriter {
	return dropstorage.NewMemoryStorageWriter(v.maxSection, v.pageSize)
}

func (v storageFactory) DepositReadOnlyWriter(bundle.SnapshotWriter) error {
	return nil // TODO
}
