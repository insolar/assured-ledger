// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

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
