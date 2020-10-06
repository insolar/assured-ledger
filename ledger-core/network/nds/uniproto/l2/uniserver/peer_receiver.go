// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type PeerReceiver struct {
	Ctx         context.Context
	PeerManager *PeerManager
	Relayer     Relayer
	HTTP        HTTPReceiverFunc
	Parser      uniproto.Parser
}

type PacketErrDetails struct {
	Header uniproto.Header
	Pulse  pulse.Number
}

type ConnErrDetails struct {
	Local, Remote nwapi.Address
}

func (p PeerReceiver) ReceiveStream(remote nwapi.Address, conn io.ReadWriteCloser, wOut l1.OneWayTransport) (runFn func() error, err error) {
	defer func() {
		_ = iokit.SafeClose(conn)
		err = throw.R(recover(), err)
	}()

	isIncoming := wOut != nil

	var (
		fn          uniproto.VerifyHeaderFunc
		peer        *Peer
		restriction TransportStreamFormat
	)

	// ATTENTION! Do all checks here - this will prevent an attacker from creation of multiple routines.

	if fn, peer, err = p.resolvePeer(remote, isIncoming, conn); err != nil {
		return nil, err
	}

	if restriction, err = peer.transport.addReceiver(conn, isIncoming); err != nil {
		return nil, err
	}

	var tlsState *tls.ConnectionState

	isHTTP := restriction.IsHTTP()
	if isIncoming {
		if tlsConn, ok := conn.(*tls.Conn); ok {
			state := tlsConn.ConnectionState()
			tlsState = &state

			switch state.NegotiatedProtocol {
			case "http/1.1":
				isHTTP = true
			case "":
			default:
				isHTTP = false
			}
		}

		if restriction.IsDefined() && restriction.IsHTTP() != isHTTP {
			return nil, errors.New("expected HTTP")
		}
	}

	c := conn
	conn = nil // don't close by defer

	if isHTTP {
		return func() error {
			defer peer.transport.removeReceiver(c)

			r := peer.transport.limitReader(c)
			w := peer.transport.limitWriter(c)
			err := p.receiveHTTPStream(remote, tlsState, nil, r, w, fn, restriction)
			return err
		}, nil
	}

	return func() error {
		defer peer.transport.removeReceiver(c)

		r := peer.transport.limitReader(c)

		for {
			packet := uniproto.ReceivedPacket{From: remote, Peer: peer}
			preRead, more, err := p.Parser.ReceivePacket(&packet, fn, r, restriction.IsUnlimited())

			switch isEOF := false; {
			case more < 0:
				switch err {
				case io.EOF:
					return nil
				case uniproto.ErrPossibleHTTPRequest:
					switch {
					case restriction.IsHTTP():
					case restriction.IsDefined():
						return err
					case !isIncoming:
						// TODO support some day outgoing HTTP format
						return throw.Violation("outgoing HTTP is not implemented")
					}

					w := peer.transport.limitWriter(c)
					return p.receiveHTTPStream(remote, tlsState, preRead, r, w, fn, restriction)
				}
			case err == io.EOF:
				isEOF = true
				fallthrough
			case err == nil:
				if wOut != nil {
					switch {
					case restriction != DetectByFirstPacket:
					case more == 0:
						restriction = BinaryLimitedLength
					default:
						restriction = BinaryUnlimitedLength
					}

					if q := peer.transport.rateQuota; q != nil {
						wOut = wOut.WithQuota(q.WriteBucket())
					}
					wOut.SetTag(int(restriction))
					// to prevent possible deadlock
					// don't worry for late additions - a closed connection will be detected and removed
					go peer.transport.addConnection(wOut)
					wOut = nil
				}

				switch {
				case packet.Header.IsForRelay():
					switch {
					case p.Relayer == nil:
						err = throw.Unsupported()
					case more == 0:
						err = p.Relayer.RelaySmallPacket(p.PeerManager, &packet.Packet, preRead)
					default:
						err = p.Relayer.RelayLargePacket(p.PeerManager, &packet.Packet, preRead, io.LimitedReader{R: r, N: more})
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
			err = p.Relayer.RelaySessionlessPacket(p.PeerManager, &packet.Packet, b)
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

		if selfVerified {
			// TODO read PK from packet and verify it
			return nil, throw.NotImplemented()
		}

		if dsv == nil {
			return nil, throw.Violation("unable to verify packet")
		}
		return dsv, nil
	}, nil
}

func (p PeerReceiver) receiveHTTPStream(remote nwapi.Address, tlsState *tls.ConnectionState,
	preRead []byte, r io.ReadCloser, w io.WriteCloser, fn uniproto.VerifyHeaderFunc, limit TransportStreamFormat,
) error {

	defer func() {
		_ = r.Close()
		_ = w.Close()
	}()

	remoteAddr := remote.String()

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

	writer := bufio.NewWriter(w)

	defer func() {
		_ = writer.Flush()
	}()

	if p.HTTP == nil {
		// TODO return a correct HTTP response
		return throw.Unsupported()
	}

	recData := &HTTPReceiver{
		PeerManager: p.PeerManager,
		Relayer:     p.Relayer,
		Parser:      p.Parser,
		VerifyFn:    fn,
		Format:      limit,
	}

	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return err
		}
		if p.Ctx != nil {
			req = req.WithContext(p.Ctx)
		}
		req.RemoteAddr = remoteAddr
		req.TLS = tlsState

		if err = p.HTTP(req, writer, recData); err != nil {
			return err
		}
	}
}

func (p PeerReceiver) getLocalHostID(supp uniproto.Supporter) uint32 {
	localID := p.PeerManager.Local().GetNodeID()
	if supp == nil {
		return uint32(localID)
	}
	return supp.FromLocalToPacket(localID)
}
