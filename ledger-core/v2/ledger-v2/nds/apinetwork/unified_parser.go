// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type UnifiedProtocolSet struct {
	Protocols         [ProtocolTypeMax + 1]ProtocolDescriptor
	SignatureSizeHint int // only a pre-allocation hint, actual size is taken from cryptkit.DataSignatureVerifier
}

func (p UnifiedProtocolSet) ReceivePacket(packet *Packet, headerFn VerifyHeaderFunc, r io.Reader, allowExcessive bool,
) (header []byte, sigLen int, more int64, err error) {

	header = make([]byte, LargePacketBaselineWithoutSignatureSize+p.SignatureSizeHint)

	more = -1
	if n, err := io.ReadFull(r, header[:LargePacketBaselineWithoutSignatureSize]); err != nil {
		return header[:n], 0, more, err
	}

	if _, err = packet.DeserializeMinFromBytes(header); err != nil {
		return header, 0, more, throw.WithDefaultSeverity(err, throw.ViolationSeverity)
	}

	err = func() error {
		if verifier, fullLen, err := p.verifyPacket(packet, headerFn, false); err != nil {
			return err
		} else {
			sigLen = verifier.GetDigestSize()

			switch {
			case packet.Header.IsExcessiveLength():
				if !allowExcessive {
					return throw.Violation("non-excessive connection")
				}
				if err := packet.VerifyExcessivePayload(verifier, &header, r); err != nil {
					return throw.WithDefaultSeverity(err, throw.ViolationSeverity)
				}
				more = int64(fullLen) - int64(len(header))
				return nil
			//case h.: // marker-delimited stream
			//	return p.receiveFlowPacket(from, packet, header, r)
			default:
				pos := len(header)

				if extra := int(fullLen) - pos; extra < 0 {
					return throw.Violation("insufficient length")
				} else {
					header = append(header, make([]byte, extra)...)
				}
				if _, err := io.ReadFull(r, header[pos:]); err != nil {
					return err
				}
				if err := packet.VerifyNonExcessivePayload(verifier, header); err != nil {
					return throw.WithDefaultSeverity(err, throw.ViolationSeverity)
				}
				more = 0
				return nil
			}
		}
	}()
	return
}

func (p UnifiedProtocolSet) ReceiveDatagram(packet *Packet, headerFn VerifyHeaderFunc, b []byte) (length, sigLen int, err error) {
	if _, err = packet.DeserializeMinFromBytes(b); err != nil {
		return -1, 0, throw.WithDefaultSeverity(err, throw.ViolationSeverity)
	}

	err = func() error {
		if verifier, fullLen, err := p.verifyPacket(packet, headerFn, true); err != nil {
			return err
		} else if err = packet.VerifyNonExcessivePayload(verifier, b); err != nil {
			return throw.WithDefaultSeverity(err, throw.ViolationSeverity)
		} else {
			length = int(fullLen)
			sigLen = verifier.GetDigestSize()
			return nil
		}
	}()
	return
}

func (p UnifiedProtocolSet) verifyPacket(packet *Packet, headerFn VerifyHeaderFunc, isDatagram bool,
) (cryptkit.DataSignatureVerifier, uint64, error) {
	h := packet.Header
	protocolDesc := &p.Protocols[h.GetProtocolType()]
	if !protocolDesc.IsSupported() {
		return nil, 0, throw.Violation("unsupported protocol")
	}
	packetDesc := protocolDesc.SupportedPackets[h.GetPacketType()]

	var (
		verifier cryptkit.DataSignatureVerifier
		fullLen  uint64
	)

	if err := func() (err error) {
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
			break
		case h.IsRelayRestricted():
			return throw.RemoteBreach("relay is restricted by source")
		case packetDesc.Flags&DisableRelay != 0:
			return throw.Violation("relay is restricted by receiver")
		}

		switch {
		case h.SourceID == 0:
			if packetDesc.Flags&NoSourceId == 0 {
				return throw.Violation("non-sourced packet")
			}
		case h.SourceID == h.ReceiverID || h.SourceID == h.TargetID:
			return throw.Violation("loopback")
		case packetDesc.Flags&NoSourceId != 0:
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
