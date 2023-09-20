package uniproto

import "github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"

// Descriptor provides information about a protocol.
// Each protocol has to define at least packet type #0.
type Descriptor struct {
	// TLS config
	// Route Validation / This ID
	// Certificate / signature? then needs unified_packet

	// Supporter is optional
	Supporter        Supporter
	// SupportedPackets is a list of packet types supported by the protocol.
	// There are some restrictions on protocol type / packet type combinations. See ProtocolType.
	SupportedPackets PacketDescriptors
}

func (d Descriptor) IsSupported() bool {
	return d.SupportedPackets[0].IsSupported()
}

// PacketDescriptor defines flags and packet size limit for a given packet type of a protocol.
type PacketDescriptor struct {
	Flags      Flags
	// LengthBits is a number of bits used to represent maximum size of a packet.
	// When LengthBits is [0..4], less than MinLengthBits, then the packet is considered as disabled.
	// When LengthBits is over ExcessiveLengthBits then this packet can't be supported yet.
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

// VerifyHeaderFunc can provide DataSignatureVerifier for the given packet.
type VerifyHeaderFunc func(h *Header, flags Flags, s Supporter) (cryptkit.DataSignatureVerifier, error)

type Descriptors [ProtocolTypeCount]Descriptor
type PacketDescriptors [PacketTypeCount]PacketDescriptor
