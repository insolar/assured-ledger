package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Parser is a helper structure to receive and check packet
type Parser struct {
	Protocols   Descriptors
	SigSizeHint int // a pre-allocation hint, MUST be >= actual size from cryptkit.DataSignatureVerifier
	Dispatcher  Dispatcher
}

func (p Parser) GetMode() ConnectionMode {
	return p.Dispatcher.GetMode()
}

func (p Parser) ReceivePacket(packet *ReceivedPacket, headerFn VerifyHeaderFunc, r io.Reader, allowExcessive bool,
) (preRead []byte, more int64, err error) {

	const minSize = LargePacketBaselineWithoutSignatureSize
	preRead = make([]byte, minSize, minSize+p.SigSizeHint)

	more = -1
	switch readLen, err := io.ReadFull(r, preRead); err {
	case nil:
		preRead = preRead[:readLen]
	case io.EOF:
		if readLen == len(preRead) {
			break
		}
		fallthrough
	default:
		return preRead[:readLen], more, err
	}

	if _, err = packet.DeserializeMinFromBytes(preRead); err != nil {
		return preRead, more, err
	}

	err = func() error {
		verifier, fullLen, err := p.verifyPacket(&packet.Packet, headerFn, false)
		if err != nil {
			return err
		}
		packet.verifier.Verifier = verifier

		switch {
		case packet.Header.IsExcessiveLength():
			if !allowExcessive {
				return throw.Violation("non-excessive connection")
			}
			if err := packet.VerifyExcessivePayload(packet.verifier, &preRead, r); err != nil {
				return err
			}
			more = int64(fullLen) - int64(len(preRead))
			return nil
		// case h.: // marker-delimited stream
		//	return p.receiveFlowPacket(from, packet, header, r)
		default:
			ofs := len(preRead)
			extra := int(fullLen) - ofs
			if extra < 0 {
				return throw.Violation("insufficient length")
			}

			preRead = append(preRead, make([]byte, extra)...)
			if _, err := io.ReadFull(r, preRead[ofs:]); err != nil {
				return err
			}
			if err := packet.VerifyNonExcessivePayload(packet.verifier, preRead); err != nil {
				return err
			}
			more = 0
			return nil
		}
	}()
	return preRead, more, err
}

func (p Parser) ReceiveDatagram(packet *ReceivedPacket, headerFn VerifyHeaderFunc, b []byte) (int, error) {
	if _, err := packet.DeserializeMinFromBytes(b); err != nil {
		return -1, err
	}

	if verifier, fullLen, err := p.verifyPacket(&packet.Packet, headerFn, true); err != nil {
		return 0, err
	} else if err = packet.VerifyNonExcessivePayload(PacketVerifier{verifier}, b); err != nil {
		return 0, err
	} else {
		packet.verifier.Verifier = verifier
		return int(fullLen), nil
	}
}

func (p Parser) verifyPacket(packet *Packet, headerFn VerifyHeaderFunc, isDatagram bool,
) (cryptkit.DataSignatureVerifier, uint64, error) {
	h := packet.Header

	var (
		verifier cryptkit.DataSignatureVerifier
		fullLen  uint64
	)

	if err := func() (err error) {
		if !p.GetMode().IsProtocolAllowed(h.GetProtocolType()) {
			return throw.Violation("protocol is disabled")
		}

		protocolDesc := &p.Protocols[h.GetProtocolType()]
		if !protocolDesc.IsSupported() {
			return throw.Violation("unsupported protocol")
		}
		packetDesc := protocolDesc.SupportedPackets[h.GetPacketType()]

		switch {
		case !packetDesc.IsSupported():
			return throw.Violation("unsupported packet")
		case !isDatagram && packetDesc.Flags&DatagramOnly != 0:
			return throw.Violation("datagram-only packet")
		case isDatagram && packetDesc.Flags&DatagramAllowed == 0:
			return throw.Violation("non-datagram packet")
		case h.TargetID == h.ReceiverID:
			if h.TargetID != 0 {
				break
			}
			fallthrough
		case h.TargetID == 0:
			if packetDesc.Flags&OptionalTarget == 0 {
				return throw.Violation("non-targeted packet")
			}
		case h.IsRelayRestricted():
			return throw.RemoteBreach("relay is restricted by source")
		case packetDesc.Flags&DisableRelay != 0:
			return throw.Violation("relay is restricted by receiver")
		}

		switch {
		case h.SourceID == 0:
			if packetDesc.Flags&NoSourceID == 0 {
				return throw.Violation("non-sourced packet")
			}
		case h.SourceID == h.ReceiverID || h.SourceID == h.TargetID:
			return throw.Violation("loopback")
		case packetDesc.Flags&NoSourceID != 0:
			return throw.Violation("sourced packet")
		}

		if fullLen, err = packet.Header.GetFullLength(); err != nil {
			return err
		}
		if !packetDesc.IsAllowedLength(fullLen) {
			return throw.Violation("length is out of limit")
		}

		if headerFn != nil {
			switch verifier, err = headerFn(&packet.Header, packetDesc.Flags, protocolDesc.Supporter); {
			case err != nil:
				return err
			case verifier == nil:
				return throw.Violation("PK is unavailable")
			}
		}
		if protocolDesc.Supporter != nil {
			verifier, err = protocolDesc.Supporter.VerifyHeader(&packet.Header, packet.PulseNumber, verifier)
		}
		return err
	}(); err != nil {
		return nil, 0, err
	}
	if verifier == nil {
		return nil, 0, throw.IllegalState()
	}

	return verifier, fullLen, nil
}
