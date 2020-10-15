// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// NoopReceiver does nothing
type NoopReceiver struct{}

func (p *NoopReceiver) Start(PeerManager)     {}
func (p *NoopReceiver) NextPulse(pulse.Range) {}
func (p *NoopReceiver) Stop()                 {}

func (p *NoopReceiver) ReceiveSmallPacket(*ReceivedPacket, []byte) {}

func (p *NoopReceiver) ReceiveLargePacket(*ReceivedPacket, []byte, io.LimitedReader) error {
	return nil
}
