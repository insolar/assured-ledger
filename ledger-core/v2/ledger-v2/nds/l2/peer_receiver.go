// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"crypto/tls"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PeerReceiver struct {
	PeerManager *PeerManager
	Protocols   *apinetwork.UnifiedProtocolSet
	ModeFn      func() ConnectionMode
}

func (p PeerReceiver) ReceiveStream(local, remote l1.Address, conn io.ReadWriteCloser, w l1.OutTransport) (runFn func() error, err error) {
	defer func() {
		_ = iokit.SafeClose(conn)
		err = throw.R(recover(), err)
	}()

	isIncoming := w != nil

	var (
		fn    apinetwork.VerifyHeaderFunc
		peer  *Peer
		limit PayloadLengthLimit
	)

	// ATTENTION! Don't run "go" before checks to prevent an attacker from
	// creation of multiple routines.

	if fn, peer, err = p.resolvePeer(remote, isIncoming, conn); err != nil {
		return nil, err
	}

	if limit, err = peer.transport.addReceiver(conn, isIncoming); err != nil {
		return nil, err
	}

	c := conn
	conn = nil // don't close by defer

	runFn = func() error {
		defer peer.transport.removeReceiver(c)

		var r io.ReadCloser
		if q := peer.transport.rateQuota; q != nil {
			r = iokit.RateLimitReader(c, q)
		} else {
			r = c
		}

		for {
			var packet apinetwork.Packet
			preRead, sigLen, more, err := p.Protocols.ReceivePacket(&packet, fn, r, limit != NonExcessivePayloadLength)

			switch isEOF := false; {
			case more < 0:
				break
			case err == io.EOF:
				isEOF = true
				fallthrough
			case err == nil:
				if w != nil {
					switch {
					case limit != DetectByFirstPayloadLength:
					case more == 0:
						limit = NonExcessivePayloadLength
					default:
						limit = UnlimitedPayloadLength
					}

					if q := peer.transport.rateQuota; q != nil {
						w = w.WithQuota(q.WriteBucket())
					}
					w.SetTag(int(limit))
					// to prevent possible deadlock
					// don't worry for late additions - a closed connection will be detected and removed
					go peer.transport.addConnection(w)
					w = nil
				}

				receiver := p.Protocols.Protocols[packet.Header.GetProtocolType()].Receiver
				if more == 0 {
					err = receiver.ReceiveSmallPacket(remote, packet, preRead, sigLen)
				} else {
					err = receiver.ReceiveLargePacket(remote, packet, preRead, sigLen, io.LimitedReader{R: r, N: more})
				}

				switch {
				case err != nil:
					//
				case isEOF:
					return nil
				case err == nil:
					continue
				}
				fallthrough
			default:
				err = throw.WithDetails(err, PacketErrDetails{packet.Header, packet.PulseNumber})
			}
			return throw.WithDetails(err, ConnErrDetails{local, remote})
		}
	}
	return runFn, nil
}

func (p PeerReceiver) ReceiveDatagram(remote l1.Address, b []byte) (err error) {

	var fn apinetwork.VerifyHeaderFunc
	fn, _, err = p.resolvePeer(remote, true, nil)

	n := -1
	sigLen := 0
	var packet apinetwork.Packet
	if n, sigLen, err = p.Protocols.ReceiveDatagram(&packet, fn, b); err == nil {
		receiver := p.Protocols.Protocols[packet.Header.GetProtocolType()].Receiver

		switch err = receiver.ReceiveSmallPacket(remote, packet, b, sigLen); {
		case err != nil:
			//
		case n != len(b):
			err = throw.Violation("data beyond length")
		default:
			return nil
		}
	}
	if n < 0 {
		return err
	}
	return throw.WithDetails(err, PacketErrDetails{packet.Header, packet.PulseNumber})
}

func (p PeerReceiver) resolvePeer(remote l1.Address, isIncoming bool, conn io.ReadWriteCloser) (apinetwork.VerifyHeaderFunc, *Peer, error) {
	var tlsConn *tls.Conn
	if t, ok := conn.(*tls.Conn); ok {
		tlsConn = t
	}

	peer, err := p.PeerManager.peerNoLocal(remote)
	switch {
	case err != nil:
		return nil, nil, err
	case peer != nil:
		return p.knownPeerVerify(peer, remote, tlsConn), peer, nil
	}

	switch {
	case !isIncoming:
		err = throw.Impossible()
	default:
		mode := p.ModeFn()
		if !mode.IsUnknownPeerAllowed() {
			err = throw.Violation("unknown peer")
			break
		}

		var fn apinetwork.VerifyHeaderFunc
		switch peer, err = p.PeerManager.connectFrom(remote, func(peer *Peer) error {
			fn = p.unknownPeerVerify(peer, remote, tlsConn)
			return nil
		}); {
		case err != nil:
			break
		case fn != nil:
			return fn, peer, nil
		default:
			return p.knownPeerVerify(peer, remote, tlsConn), peer, nil
		}
	}
	return nil, nil, err
}

func (p PeerReceiver) knownPeerVerify(peer *Peer, remote l1.Address, tlsConn *tls.Conn) apinetwork.VerifyHeaderFunc {
	tlsState := 0
	switch {
	case tlsConn == nil:
		//
	case peer.VerifyByTls(tlsConn):
		tlsState = 1
	default:
		tlsState = -1
	}

	return func(supp apinetwork.ProtocolSupporter, desc apinetwork.ProtocolPacketDescriptor,
		header *apinetwork.Header, pn pulse.Number,
	) (cryptkit.DataSignatureVerifier, error) {

		if desc.TransportFlags&apinetwork.OmitSignatureOverTls != 0 {
			if tlsState < 0 {
				return nil, throw.RemoteBreach("TLS check failed")
			}
			// TODO support header/content validation without signature field when TLS is available by providing zero len hasher/verifier
			return nil, throw.NotImplemented()
		}

		// TODO check receiver
		// TODO change peer for relay

		switch dsv, err := peer.getVerifier(supp, header, p.PeerManager.sigvFactory); {
		case err != nil:
			return nil, err
		case dsv == nil:
			return nil, throw.Violation("PK is not available")
		default:
			return dsv, nil
		}
	}
}

// LOCK: WARNING! This method is called under PeerManager's lock
func (p PeerReceiver) unknownPeerVerify(peer *Peer, remote l1.Address, tlsConn *tls.Conn) apinetwork.VerifyHeaderFunc {

	// identify peer by TLS
	// identify peer by IP
	// check if it was redirected

	initFn := func(supp apinetwork.ProtocolSupporter, desc apinetwork.ProtocolPacketDescriptor,
		header *apinetwork.Header, number pulse.Number,
	) (apinetwork.VerifyHeaderFunc, error) {

		// TODO problem - IP can be unknown, but SourceID correct and matching PK/signature, then the peer has to be changed ...

		return p.knownPeerVerify(peer, remote, tlsConn), nil
	}

	var knownFn apinetwork.VerifyHeaderFunc

	return func(supp apinetwork.ProtocolSupporter, desc apinetwork.ProtocolPacketDescriptor,
		header *apinetwork.Header, number pulse.Number,
	) (cryptkit.DataSignatureVerifier, error) {
		if initFn != nil {
			var err error
			if knownFn, err = initFn(supp, desc, header, number); err != nil {
				return nil, err
			}
			initFn = nil
		}
		return knownFn(supp, desc, header, number)
	}
}
