// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
)

type Relayer interface {
	RelaySessionlessPacket(*uniproto.Packet, []byte) error
	RelaySmallPacket(*uniproto.Packet, []byte) error
	RelayLargePacket(*uniproto.Packet, []byte, io.LimitedReader) error
}
