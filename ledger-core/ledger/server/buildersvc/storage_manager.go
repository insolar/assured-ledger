// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func newStorageWriter(pn pulse.Number) *storageManager {
	return &storageManager{
		writeChan: make(chan writeBundle, 32),
		oobChan: make(chan oobEvent, 2),
	}
}

type storageManager struct {
	writeChan chan writeBundle // close on pulse change
	oobChan   chan oobEvent    // close on pulse change
}

type oobEvent struct {
	// suspend & get NSH
	// resume
}



type writeBundle struct {
	jetDrop jet.DropID
	bundle  lineage.BundleResolver
	future  *Future
}

type DropStorageManager interface {
	CreateBuilder(pulse.Range) PulseDropBuilder
//	CreateGap()
//  CreateMissed()
}

type PulseDropBuilder interface {
	Layout(*jet.Tree, census.OnlinePopulation)
	// Sections()
	Drops([]jet.LegID)
	Build() DropBuilder
}

type DropBuilder interface {

}

type JetDropBuilder interface {
	AddRecords([]Record) (bool, uint32, error)
}

type Record struct {
	RecordRef reference.Holder
	rms.CatalogEntry
	Body    longbits.ByteString
	Payload longbits.ByteString
	Extensions []RecordExtension
}

type RecordExtension struct {
	ExtensionID uint32
	Content longbits.ByteString
}

type SectionID uint16
type ChapterID uint32
type StorageLocator uint64

func (v StorageLocator) SectionID() SectionID {
	return SectionID(v >> 48)
}

func (v StorageLocator) ChapterID() ChapterID {
	return ChapterID(v >> 24) & 0x00FF_FFFF
}

func (v StorageLocator) ChapterOffset() uint32 {
	return uint32(v) & 0x00FF_FFFF
}

func (v StorageLocator) Offset() uint64 {
	return uint64(v) & 0xFFFF_FFFF_FFFF
}
