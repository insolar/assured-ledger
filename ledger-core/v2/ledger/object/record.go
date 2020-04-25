// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// TypeID encodes a record object type.
type TypeID uint32

// TypeIDSize is a size of TypeID type.
const TypeIDSize = 4

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/object.RecordCleaner -o ./ -s _mock.go -g

// RecordCleaner provides an interface for removing records from a storage.
type RecordCleaner interface {
	// DeleteForPN method removes records from a storage for a pulse
	DeleteForPN(ctx context.Context, pulse insolar.PulseNumber)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/object.RecordPositionAccessor -o ./ -s _mock.go

// RecordPositionAccessor provides an interface for fetcing position of records.
type RecordPositionAccessor interface {
	// LastKnownPosition returns last known position of record in Pulse.
	LastKnownPosition(pn insolar.PulseNumber) (uint32, error)
	// AtPosition returns record ID for a specific pulse and a position
	AtPosition(pn insolar.PulseNumber, position uint32) (insolar.ID, error)
}
