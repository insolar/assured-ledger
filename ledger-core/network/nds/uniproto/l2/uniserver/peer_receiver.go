// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type PeerReceiver struct {
	PeerManager *PeerManager
	Relayer     Relayer
	Parser      uniproto.Parser
}

type PacketErrDetails struct {
	Header uniproto.Header
	Pulse  pulse.Number
}

type ConnErrDetails struct {
	Local, Remote nwapi.Address
}

func (p PeerReceiver) ReceiveStream(remote nwapi.Address, conn io.ReadWriteCloser, w l1.OutTransport) (runFn func() error, err error) {
	defer func() {
		_ = iokit.SafeClose(conn)
		err = throw.R(recover(), err)
	}()

	isIncoming := w != nil

	var (
		fn    uniproto.VerifyHeaderFunc
		peer  *Peer
		limit TransportStreamFormat
	)

	// ATTENTION! Don't run "go" before checks - this will prevent an attacker from creation of multiple routines.

	if fn, peer, err = p.resolvePeer(remote, isIncoming, conn); err != nil {
		return nil, err
	}

	if limit, err = peer.transport.addReceiver(conn, isIncoming); err != nil {
		return nil, err
	}

	isHTTP := limit.IsHTTP()
	if isIncoming {
		if tlsConn, ok := conn.(*tls.Conn); ok {
			state := tlsConn.ConnectionState()
			switch state.NegotiatedProtocol {
			case "http/1.1":
				isHTTP = true
			case "":
			default:
				isHTTP = false
			}
		}

		if limit.IsDefined() && limit.IsHTTP() != isHTTP {
			return nil, errors.New("expected HTTP")
		}
	}

	c := conn
	conn = nil // don't close by defer

	return func() error {
		defer peer.transport.removeReceiver(c)

		var r io.ReadCloser
		if q := peer.transport.rateQuota; q != nil {
			r = iokit.RateLimitReader(c, q)
		} else {
			r = c
		}

		if isHTTP {
			return p.receiveHTTPStream(nil, r, fn, limit)
		}

		for {
			packet := uniproto.ReceivedPacket{From: remote, Peer: peer}
			preRead, more, err := p.Parser.ReceivePacket(&packet, fn, r, limit.IsUnlimited())

			switch isEOF := false; {
			case more < 0:
				switch err {
				case io.EOF:
					return nil
				case uniproto.ErrPossibleHTTPRequest:
					if !limit.IsDefined() || limit.IsHTTP() {
						return p.receiveHTTPStream(preRead, r, fn, limit)
					}
				}
			case err == io.EOF:
				isEOF = true
				fallthrough
			case err == nil:
				if w != nil {
					switch {
					case limit != DetectByFirstPacket:
					case more == 0:
						limit = BinaryLimitedLength
					default:
						limit = BinaryUnlimitedLength
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

				switch {
				case packet.Header.IsForRelay():
					switch {
					case p.Relayer == nil:
						err = throw.Unsupported()
					case more == 0:
						err = p.Relayer.RelaySmallPacket(&packet.Packet, preRead)
					default:
						err = p.Relayer.RelayLargePacket(&packet.Packet, preRead, io.LimitedReader{R: r, N: more})
					}
				case packet.Header.IsBodyEncrypted():
					packet.Decrypter, err = p.PeerManager.GetLocalDataDecrypter()
					if err != nil {
						break
					}
					fallthrough
				default:
					receiver := p.Parser.Dispatcher.GetReceiver(packet.Header.GetProtocolType())
					if more == 0 {
						receiver.ReceiveSmallPacket(&packet, preRead)
					} else {
						err = receiver.ReceiveLargePacket(&packet, preRead, io.LimitedReader{R: r, N: more})
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
	}, nil
}

func (p PeerReceiver) ReceiveDatagram(remote nwapi.Address, b []byte) (err error) {

	var (
		fn   uniproto.VerifyHeaderFunc
		peer *Peer
	)
	fn, peer, err = p.resolvePeer(remote, true, nil)
	if err != nil {
		return err
	}

	var n int
	packet := uniproto.ReceivedPacket{From: remote, Peer: peer}
	if n, err = p.Parser.ReceiveDatagram(&packet, fn, b); err == nil {
		switch {
		case !packet.Header.IsForRelay():
			receiver := p.Parser.Dispatcher.GetReceiver(packet.Header.GetProtocolType())
			receiver.ReceiveSmallPacket(&packet, b)
		case p.Relayer == nil:
			err = throw.Unsupported()
		default:
			err = p.Relayer.RelaySessionlessPacket(&packet.Packet, b)
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

func (p PeerReceiver) resolvePeer(remote nwapi.Address, isIncoming bool, conn io.ReadWriteCloser) (uniproto.VerifyHeaderFunc, *Peer, error) {
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
			if !p.Parser.GetMode().IsUnknownPeerAllowed() {
				return throw.Violation("unknown peer")
			}
			isNew = true
			return nil
		})
	}
	if err == nil {
		var fn uniproto.VerifyHeaderFunc
		if fn, err = p.checkPeer(peer, tlsConn); err == nil {
			if isNew {
				peer.UpgradeState(Connected)
			}
			return fn, peer, nil
		}
	}
	return nil, nil, err
}

func toHostID(id uint32, supp uniproto.Supporter) nwapi.HostID {
	switch {
	case id == 0:
		return 0
	case supp == nil:
		return nwapi.HostID(id)
	default:
		return supp.FromPacketToHostID(id)
	}
}

func (p PeerReceiver) checkSourceAndReceiver(peer *Peer, supp uniproto.Supporter, header *uniproto.Header,
) (selfVerified bool, dsv cryptkit.DataSignatureVerifier, err error) {

	if err = func() (err error) {
		if header.ReceiverID != 0 {
			// ReceiverID must match Local
			if rid := toHostID(header.ReceiverID, supp); !p.isLocalHostID(rid) {
				return throw.RemoteBreach("wrong ReceiverID")
			}
		} else {
			header.ReceiverID = p.getLocalHostID(supp)
		}

		switch {
		case header.SourceID != 0:
			if header.IsRelayRestricted() || header.IsForRelay() {
				// SourceID must match Peer
				// Signature must match Peer
				if sid := toHostID(header.SourceID, supp); p.hasHostID(sid, peer) {
					dsv, err = peer.GetDataVerifier()
				} else {
					return throw.RemoteBreach("wrong SourceID")
				}
			} else {
				// Peer must be known and validated
				// Signature must match SourceID
				if err = peer.checkVerified(); err != nil {
					return err
				}

				sid := toHostID(header.SourceID, supp)
				if peer, err = p.PeerManager.peerNotLocal(nwapi.NewHostID(sid)); err == nil {
					dsv, err = peer.GetDataVerifier()
				}
			}
		case header.IsRelayRestricted():
			// Signature must match Peer
			dsv, err = peer.GetDataVerifier()
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
	return selfVerified, dsv, err
}

func (p PeerReceiver) hasHostID(id nwapi.HostID, peer *Peer) bool {
	_, pr := p.PeerManager.peer(nwapi.NewHostID(id))
	return peer == pr
}

func (p PeerReceiver) isLocalHostID(id nwapi.HostID) bool {
	idx, pr := p.PeerManager.peer(nwapi.NewHostID(id))
	return idx == 0 && pr != nil
}

func (p PeerReceiver) checkTarget(supp uniproto.Supporter, header *uniproto.Header) (relayTo *Peer, err error) {
	switch {
	case !header.IsTargeted():
		return nil, nil
	case header.IsForRelay():
		tid := toHostID(header.TargetID, supp)
		relayTo, err = p.PeerManager.peerNotLocal(nwapi.NewHostID(tid))
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
		if tid := toHostID(header.TargetID, supp); !p.isLocalHostID(tid) {
			return nil, throw.RemoteBreach("wrong TargetID")
		}
		return nil, err
	}
}

func (p PeerReceiver) checkPeer(peer *Peer, tlsConn *tls.Conn) (uniproto.VerifyHeaderFunc, error) {
	tlsStatus := 0 // TLS is not present
	if tlsConn != nil {
		switch ok, err := peer.verifyByTLS(tlsConn); {
		case err != nil:
			return nil, err
		case ok:
			tlsStatus = 1 // TLS check was ok
		default:
			tlsStatus = -1 // unable to match TLS - it doesn't mean failure yet!
		}
	}

	return func(header *uniproto.Header, flags uniproto.Flags, supp uniproto.Supporter) (dsv cryptkit.DataSignatureVerifier, err error) {

		selfVerified := false
		if selfVerified, dsv, err = p.checkSourceAndReceiver(peer, supp, header); err != nil {
			return nil, err
		}
		if selfVerified && flags&uniproto.SourcePK == 0 {
			// requires support of self-verified packets (packet must have a PK field)
			return nil, throw.Violation("must have source PK")
		}

		switch relayTo, err := p.checkTarget(supp, header); {
		case err != nil:
			return nil, err
		case relayTo != nil:
			// Relayed packet must always have a signature of source hence OmitSignatureOverTLS is ignored
			// Actual relay operation will be performed after packet parsing
		default:
			if tlsStatus != 0 && flags&uniproto.OmitSignatureOverTLS != 0 {
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

func (p PeerReceiver) receiveHTTPStream(preRead []byte, r io.ReadCloser, fn uniproto.VerifyHeaderFunc, limit TransportStreamFormat) error {
	defer func() {
		_ = r.Close()
	}()

	var reader *bufio.Reader
	if len(preRead) > 0 {
		switch string(preRead[:5]) {
		case "GET /", "PUT /", "HEAD ", "POST ":
		default:
			return throw.Violation("unsupported header")
		}
		reader = bufio.NewReader(iokit.PrependReader(preRead, r))
	} else {
		reader = bufio.NewReader(r)
	}

	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return err
		}
		err = p.processHTTP(req, fn, limit)
		if err != nil {
			return err
		}
	}
}

func (p PeerReceiver) processHTTP(req *http.Request, _ uniproto.VerifyHeaderFunc, _ TransportStreamFormat) error {
	// TODO Read packet from HTTP body
	runtime.KeepAlive(req)
	return throw.NotImplemented()
}

func (p PeerReceiver) getLocalHostID(supp uniproto.Supporter) uint32 {
	localID := p.PeerManager.Local().GetNodeID()
	if supp == nil {
		return uint32(localID)
	}
	return supp.FromLocalToPacket(localID)
}
