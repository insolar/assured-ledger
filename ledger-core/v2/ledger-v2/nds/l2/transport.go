// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

// Transport hides pooling logic etc
type Transport interface {
	Send(to Address, payload io.WriterTo) error
	Capabilities() TransportCapabilities
}

type TransportReceiveFunc func(from Address, payload longbits.ByteString)

type TransportCapabilities uint8

const (
	LargeDataTransport TransportCapabilities = 1 << iota
)
