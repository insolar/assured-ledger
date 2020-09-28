// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import "github.com/insolar/assured-ledger/ledger-core/network/nwapi"

var _ nwapi.DeserializationContext = packetContext{}

type packetContext struct {
	pf nwapi.DeserializationFactory
}

func (v packetContext) GetPayloadFactory() nwapi.DeserializationFactory {
	return v.pf
}
