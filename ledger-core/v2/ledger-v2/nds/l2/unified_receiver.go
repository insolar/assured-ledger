// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type UnifiedReceiver interface {
	LookupAddress(remote net.Addr) (l1.Address, apinetwork.VerifyHeaderFunc, error)
	// ReceiveSmallPacket is called on small (non-excessive length) packets, (b) is exactly whole packet
	ReceiveSmallPacket(from l1.Address, packet apinetwork.Packet, b []byte) error
	// ReceiveLargePacket is called on large (excessive length) packets, (preRead) is a pre-read portion, that can be larger than a header, and (r) is configured for the remaining length.
	ReceiveLargePacket(from l1.Address, packet apinetwork.Packet, preRead []byte, r io.LimitedReader) error
	// ReceiveError is called on errors
	ReceiveError(local, remote net.Addr, err error)
}

type UnifiedReceiveHelper struct {
	Protocols *apinetwork.UnifiedProtocolSet
	Receiver  UnifiedReceiver
}

type ConnErrDetails struct {
	Local, Remote net.Addr
}

type PacketErrDetails struct {
	Header apinetwork.Header
	Pulse  pulse.Number
}

func (v UnifiedReceiveHelper) ReceiveStream(local, remote net.Addr, r io.ReadCloser, limit PayloadLengthLimit, readyFn StreamReadyFunc) {
	defer func() {
		_ = r.Close()
		if err := throw.R(recover(), nil); err != nil {
			v.Receiver.ReceiveError(local, remote, err)
		}
	}()

	from, fn, err := v.Receiver.LookupAddress(remote)
	if err != nil {
		v.Receiver.ReceiveError(local, remote, err)
		return
	}

	isFirst := true

	for {
		var packet apinetwork.Packet
		preRead, more, err := v.Protocols.ReceivePacket(&packet, fn, r, limit != NonExcessivePayloadLength)
		switch {
		case more < 0:
			// header wasn't read, so we can't add PacketErrDetails
		case err == nil || err == io.EOF:
			if isFirst {
				switch isFirst = false; {
				case limit != DetectByFirstPayloadLength:
					//
				case more == 0:
					limit = NonExcessivePayloadLength
				default:
					limit = UnlimitedPayloadLength
				}
				if readyFn != nil {
					readyFn(r, limit)
				}
			}

			if more == 0 {
				err = v.Receiver.ReceiveSmallPacket(from, packet, preRead)
			} else {
				err = v.Receiver.ReceiveLargePacket(from, packet, preRead, io.LimitedReader{R: r, N: more})
			}

			if err == nil {
				// io.EOF will be handled by inability to read header (more < 0)
				continue
			}
			fallthrough
		default:
			err = throw.WithDetails(err, PacketErrDetails{packet.Header, packet.PulseNumber})
		}
		v.Receiver.ReceiveError(local, remote, err)
		return
	}
}

func (v UnifiedReceiveHelper) ReceiveDatagram(local, remote net.Addr, b []byte) {
	defer func() {
		if err := throw.R(recover(), nil); err != nil {
			v.Receiver.ReceiveError(local, remote, err)
		}
	}()

	from, fn, err := v.Receiver.LookupAddress(remote)
	if err == nil {
		n := -1
		var packet apinetwork.Packet
		if n, err = v.Protocols.ReceiveDatagram(&packet, fn, b); err == nil {
			switch err = v.Receiver.ReceiveSmallPacket(from, packet, b); {
			case err != nil:
				//
			case n != len(b):
				err = throw.Violation("data beyond length")
			default:
				return
			}
		}
		if n > 0 {
			err = throw.WithDetails(err, PacketErrDetails{packet.Header, packet.PulseNumber})
		}
	}
	v.Receiver.ReceiveError(local, remote, err)
}

func (v UnifiedReceiveHelper) ReceiveError(local l1.Address, remote l1.Address, err error) {
	v.Receiver.ReceiveError(local, remote, err)
}
