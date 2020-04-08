// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type VerifyHeaderFunc func(h *Header, flags ProtocolFlags, s ProtocolSupporter) (cryptkit.DataSignatureVerifier, error)

type ProtocolSupporter interface {
	VerifyHeader(*Header, pulse.Number, cryptkit.DataSignatureVerifier) (cryptkit.DataSignatureVerifier, error)
	ToHostId(id uint32) HostId
}

type ProtocolReceiver interface {
	// ReceiveSmallPacket is called on small (non-excessive length) packets, (b) is exactly whole packet
	ReceiveSmallPacket(from Address, packet Packet, b []byte, signatureLen int) error
	// ReceiveLargePacket is called on large (excessive length) packets, (preRead) is a pre-read portion, that can be larger than a header, and (r) is configured for the remaining length.
	ReceiveLargePacket(from Address, packet Packet, preRead []byte, signatureLen int, r io.LimitedReader) error
}

type ProtocolReceivers = [ProtocolTypeMax + 1]ProtocolReceiver

type ProtocolDescriptor struct {
	// TLS config
	// Route Validation / This ID
	// Certificate / signature? then needs unified_packet
	Receiver         ProtocolReceiver
	Supporter        ProtocolSupporter
	SupportedPackets [PacketTypeMax + 1]ProtocolPacketDescriptor
}

func (d ProtocolDescriptor) IsSupported() bool {
	return d.Receiver != nil
}

type ProtocolPacketDescriptor struct {
	Flags      ProtocolFlags
	LengthBits uint8
}

func (d ProtocolPacketDescriptor) IsSupported() bool {
	return d.LengthBits >= MinLengthBits
}

func (d ProtocolPacketDescriptor) IsAllowedLength(fullLen uint64) bool {
	if d.LengthBits > ExcessiveLengthBits {
		return false // not yet supported
	}
	return fullLen < uint64(1)<<d.LengthBits
}

type ProtocolFlags uint8

const (
	DatagramOnly ProtocolFlags = 1 << iota
	DatagramAllowed
	DisableRelay
	OmitSignatureOverTls
	SourcePK
	OptionalTarget
	NoSourceId
)
