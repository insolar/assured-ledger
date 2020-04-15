// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type SerializationContext interface {
	nwapi.SerializationContext
	PrepareHeader(*Header, pulse.Number) (pulse.Number, error)
	VerifyHeader(*Header, pulse.Number) error
}
