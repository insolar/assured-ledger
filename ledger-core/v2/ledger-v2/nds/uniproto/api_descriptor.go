// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"

type Descriptor struct {
	// TLS config
	// Route Validation / This ID
	// Certificate / signature? then needs unified_packet
	Supporter        Supporter
	SupportedPackets PacketDescriptors
}

func (d Descriptor) IsSupported() bool {
	return d.SupportedPackets[0].IsSupported()
}

type PacketDescriptor struct {
	Flags      Flags
	LengthBits uint8
}

func (d PacketDescriptor) IsSupported() bool {
	return d.LengthBits >= MinLengthBits
}

func (d PacketDescriptor) IsAllowedLength(fullLen uint64) bool {
	if d.LengthBits > ExcessiveLengthBits {
		return false // not yet supported
	}
	return fullLen < uint64(1)<<d.LengthBits
}

type VerifyHeaderFunc func(h *Header, flags Flags, s Supporter) (cryptkit.DataSignatureVerifier, error)

type Descriptors [ProtocolTypeCount]Descriptor
type PacketDescriptors [PacketTypeCount]PacketDescriptor
