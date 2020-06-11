// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

// aliases for more convenient using in .proto file (no need to use full path)
// because alias in the same package
type MessageContext = rms.MessageContext

func RegisterMessageType(id uint64, special string, t proto.Message) {
	rms.RegisterMessageType(id, special, t)
}

type Reference = reference.Global
type LocalReference = reference.Local

type PulseNumber = pulse.Number
