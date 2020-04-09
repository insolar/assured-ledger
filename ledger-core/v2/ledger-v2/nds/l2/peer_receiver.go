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

type PacketErrDetails struct {
	Header apinetwork.Header
	Pulse  pulse.Number
}

type ConnErrDetails struct {
	Local, Remote apinetwork.Address
}

func (p PeerReceiver) ReceiveStream(remote apinetwork.Address, conn io.ReadWriteCloser, w l1.OutTransport) (runFn func() error, err error) {
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

	// ATTENTION! Don't run "go" before checks - this will prevent an attacker from creation of multiple routines.

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
			var verifier apinetwork.PacketDataVerifier
			preRead, more, err := p.Protocols.ReceivePacket(&packet, &verifier, fn, r, limit != NonExcessivePayloadLength)

			switch isEOF := false; {
			case more < 0:
				if err == io.EOF {
					return nil
				}
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

				if packet.Header.IsForRelay() {
					// TODO relay via sessionful
					err = throw.NotImplemented()
				} else {
					receiver := p.Protocols.Protocols[packet.Header.GetProtocolType()].Receiver
					if more == 0 {
						receiver.ReceiveSmallPacket(remote, packet, preRead, uint32(verifier.Verifier.GetDigestSize()))
					} else {
						err = receiver.ReceiveLargePacket(remote, packet, preRead, io.LimitedReader{R: r, N: more}, verifier)
					}
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
			return err
		}
	}
	return runFn, nil
}

func (p PeerReceiver) ReceiveDatagram(remote apinetwork.Address, b []byte) (err error) {

	var fn apinetwork.VerifyHeaderFunc
	fn, _, err = p.resolvePeer(remote, true, nil)

	n := -1
	sigLen := 0
	var packet apinetwork.Packet
	if n, sigLen, err = p.Protocols.ReceiveDatagram(&packet, fn, b); err == nil {
		if !packet.Header.IsForRelay() {
			receiver := p.Protocols.Protocols[packet.Header.GetProtocolType()].Receiver
			receiver.ReceiveSmallPacket(remote, packet, b, uint32(sigLen))
		} else {
			// TODO relay via sessionless
			err = throw.NotImplemented()
		}

		switch {
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

func (p PeerReceiver) resolvePeer(remote apinetwork.Address, isIncoming bool, conn io.ReadWriteCloser) (apinetwork.VerifyHeaderFunc, *Peer, error) {
	var tlsConn *tls.Conn
	if t, ok := conn.(*tls.Conn); ok {
		tlsConn = t
	}

	isNew := false
	peer, err := p.PeerManager.peerNotLocal(remote)
	switch {
	case err != nil:
		return nil, nil, err
	case peer != nil:
		//
	case !isIncoming:
		err = throw.Impossible()
	default:
		peer, err = p.PeerManager.connectionFrom(remote, func(*Peer) error {
			if !p.ModeFn().IsUnknownPeerAllowed() {
				return throw.Violation("unknown peer")
			}
			isNew = true
			return nil
		})
	}
	if err == nil {
		var fn apinetwork.VerifyHeaderFunc
		if fn, err = p.checkPeer(peer, tlsConn); err == nil {
			if isNew {
				peer.UpgradeState(Connected)
			}
			return fn, peer, nil
		}
	}
	return nil, nil, err
}

func toHostId(id uint32, supp apinetwork.ProtocolSupporter) apinetwork.HostId {
	switch {
	case id == 0:
		return 0
	case supp == nil:
		return apinetwork.HostId(id)
	default:
		return supp.ToHostId(id)
	}
}

func (p PeerReceiver) checkSourceAndReceiver(peer *Peer, supp apinetwork.ProtocolSupporter, header *apinetwork.Header,
) (selfVerified bool, dsv cryptkit.DataSignatureVerifier, err error) {

	if err = func() (err error) {
		if header.ReceiverID != 0 {
			// ReceiverID must match Local
			if rid := toHostId(header.ReceiverID, supp); !p.isLocalHostId(rid) {
				return throw.RemoteBreach("wrong ReceiverID")
			}
		}

		switch {
		case header.SourceID != 0:
			if header.IsRelayRestricted() || header.IsForRelay() {
				// SourceID must match Peer
				// Signature must match Peer
				if sid := toHostId(header.SourceID, supp); p.hasHostId(sid, peer) {
					dsv, err = peer.GetSignatureVerifier(p.PeerManager.svFactory)
				} else {
					return throw.RemoteBreach("wrong SourceID")
				}
			} else {
				// Peer must be known and validated
				// Signature must match SourceID
				if err = peer.checkVerified(); err != nil {
					return err
				}

				sid := toHostId(header.SourceID, supp)
				if peer, err = p.PeerManager.peerNotLocal(apinetwork.NewHostId(sid)); err == nil {
					dsv, err = peer.GetSignatureVerifier(p.PeerManager.svFactory)
				}
			}
		case header.IsRelayRestricted():
			// Signature must match Peer
			dsv, err = peer.GetSignatureVerifier(p.PeerManager.svFactory)
		default:
			// Peer must be known and validated
			// Packet must be self-validated
			if err = peer.checkVerified(); err != nil {
				return err
			}
			selfVerified = true
		}
		return
	}(); err != nil {
		return false, nil, err
	}
	return
}

func (p PeerReceiver) hasHostId(id apinetwork.HostId, peer *Peer) bool {
	_, pr := p.PeerManager.peer(apinetwork.NewHostId(id))
	return peer == pr
}

func (p PeerReceiver) isLocalHostId(id apinetwork.HostId) bool {
	idx, pr := p.PeerManager.peer(apinetwork.NewHostId(id))
	return idx == 0 && pr != nil
}

func (p PeerReceiver) checkTarget(supp apinetwork.ProtocolSupporter, header *apinetwork.Header) (relayTo *Peer, err error) {
	switch {
	case !header.IsTargeted():
		return nil, nil
	case header.IsForRelay():
		tid := toHostId(header.TargetID, supp)
		relayTo, err = p.PeerManager.peerNotLocal(apinetwork.NewHostId(tid))
		switch {
		case err != nil:
			//
		case relayTo == nil:
			err = throw.E("unknown target")
		default:
			err = relayTo.checkVerified()
		}
		return
	default:
		// TargetID must match Local
		if tid := toHostId(header.TargetID, supp); !p.isLocalHostId(tid) {
			return nil, throw.RemoteBreach("wrong TargetID")
		}
		return nil, err
	}
}

func (p PeerReceiver) checkPeer(peer *Peer, tlsConn *tls.Conn) (apinetwork.VerifyHeaderFunc, error) {
	tlsStatus := 0 // TLS is not present
	if tlsConn != nil {
		switch ok, err := peer.verifyByTls(tlsConn); {
		case err != nil:
			return nil, err
		case ok:
			tlsStatus = 1 // TLS check was ok
		default:
			tlsStatus = -1 // unable to match TLS - it doesn't mean failure yet!
		}
	}

	return func(header *apinetwork.Header, flags apinetwork.ProtocolFlags, supp apinetwork.ProtocolSupporter) (dsv cryptkit.DataSignatureVerifier, err error) {

		if !p.ModeFn().IsProtocolAllowed(header.GetProtocolType()) {
			return nil, throw.Violation("protocol is disabled")
		}

		selfVerified := false
		if selfVerified, dsv, err = p.checkSourceAndReceiver(peer, supp, header); err != nil {
			return nil, err
		}
		if selfVerified && flags&apinetwork.SourcePK == 0 {
			// requires support of self-verified packets (packet must have a PK field)
			return nil, throw.Violation("must have source PK")
		}

		switch relayTo, err := p.checkTarget(supp, header); {
		case err != nil:
			return nil, err
		case relayTo != nil:
			// Relayed packet must always have a signature of source hence OmitSignatureOverTls is ignored
			// Actual relay operation will be performed after packet parsing
		default:
			if tlsStatus != 0 && flags&apinetwork.OmitSignatureOverTls != 0 {
				if tlsStatus < 0 {
					return nil, throw.RemoteBreach("unidentified TLS cert")
				}
				// TODO support header/content validation without signature field when TLS is available by providing zero len hasher/verifier
				return nil, throw.NotImplemented()
			}
		}

		if dsv == nil {
			return nil, throw.Violation("unable to verify packet")
		}
		return dsv, nil
	}, nil
}
