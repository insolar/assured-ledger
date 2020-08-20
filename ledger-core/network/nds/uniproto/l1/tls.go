// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func NewTLS(binding nwapi.Address, config *tls.Config) SessionfulTransportProvider {
	return &TLSTransport{addr: binding.AsTCPAddr(), config: config}
}

func NewTLSTransport(binding nwapi.Address, config *tls.Config) TLSTransport {
	return TLSTransport{addr: binding.AsTCPAddr(), config: config}
}

type TLSTransport struct {
	config    *tls.Config
	addr      net.TCPAddr
	conn      net.Listener
	receiveFn SessionfulConnectFunc
}

func (p *TLSTransport) IsZero() bool {
	return p.conn == nil && p.addr.IP == nil
}

func (p *TLSTransport) CreateListeningFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case p.conn != nil:
		return nil, throw.IllegalState()
	case p.addr.IP == nil:
		return nil, throw.IllegalState()
	case len(p.config.Certificates) > 0 || p.config.GetCertificate != nil || p.config.GetConfigForClient != nil:
		// ok
	default:
		// mimics tls.CreateListeningFactory
		return nil, errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
	}

	conn, err := net.ListenTCP("tcp", &p.addr)
	if err != nil {
		return nil, err
	}
	p.conn = tls.NewListener(conn, p.config)
	p.receiveFn = receiveFn
	go runTCPListener(p.conn, p.tlsConnect)
	return p, nil
}

func (p *TLSTransport) CreateOutgoingOnlyFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	return &TLSTransport{p.config, p.addr, nil, receiveFn}, nil
}

func (p *TLSTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	return p.conn.Close()
}

func (p *TLSTransport) LocalAddr() nwapi.Address {
	if p.conn != nil {
		return nwapi.AsAddress(p.conn.Addr())
	}
	return nwapi.AsAddress(&p.addr)
}

func (p *TLSTransport) ConnectTo(to nwapi.Address, preference nwapi.Preference) (OutTransport, error) {
	return p.ConnectToExt(to, preference, nil)
}

func (p *TLSTransport) ConnectToExt(to nwapi.Address, preference nwapi.Preference, peerVerify VerifyPeerCertificateFunc) (OutTransport, error) {
	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver, preference)
	if err != nil {
		return nil, err
	}

	peerConfig := p.config
	if peerVerify != nil {
		cfg := p.config.Clone()
		cfg.VerifyPeerCertificate = peerVerify
		peerConfig = cfg
	}

	local := p.addr
	local.Port = 0

	var conn *tls.Conn
	if conn, err = tls.DialWithDialer(&net.Dialer{LocalAddr: &local}, "tcp", to.String(), peerConfig); err != nil {
		return nil, err
	}

	// force connection setup now
	if err = conn.Handshake(); err != nil {
		return nil, err
	} else if err = p.checkProtos(conn); err != nil {
		return nil, err
	}

	tcpOut := tcpOutTransport{conn, nil, 0}

	if p.receiveFn == nil {
		return &tcpOut, nil
	}
	return &tcpSemiTransport{tcpOut, p.receiveFn}, nil
}

func (p *TLSTransport) tlsConnect(local, remote nwapi.Address, conn io.ReadWriteCloser, w OutTransport, err error) bool {
	if err != nil {
		return p.receiveFn(local, remote, conn, w, err)
	}

	tlsConn := conn.(*tls.Conn)

	if err = tlsConn.Handshake(); err != nil {
		// If the handshake failed due to the client not speaking
		// TLS, assume they're speaking plaintext HTTP and write a
		// 400 response on the TLS conn's underlying net.TwoWayConn.
		if re, ok := err.(tls.RecordHeaderError); ok && re.Conn != nil && tlsRecordHeaderLooksLikeHTTP(re.RecordHeader) {
			_, _ = io.WriteString(re.Conn, "HTTP/1.0 400 Bad Request\r\n\r\nClient sent an HTTP request to an HTTPS server.\n")
			_ = re.Conn.Close()
			return true
		}
	} else if err = p.checkProtos(tlsConn); err == nil {
		return p.receiveFn(local, remote, conn, w, nil)
	}

	_ = conn.Close()

	//err = throw.WithDetails(err, l2.ConnErrDetails{Local:local, Remote:remote})
	return p.receiveFn(local, remote, nil, nil, err)
}

func (p *TLSTransport) checkProtos(tlsConn *tls.Conn) error {
	if len(p.config.NextProtos) == 0 {
		return nil
	}
	tlsState := tlsConn.ConnectionState()
	for _, proto := range p.config.NextProtos {
		if proto == tlsState.NegotiatedProtocol {
			return nil
		}
	}
	return errors.New("unmatched TLS application level protocol")
}

// tlsRecordHeaderLooksLikeHTTP reports whether a TLS record header
// looks like it might've been a misdirected plaintext HTTP request.
func tlsRecordHeaderLooksLikeHTTP(hdr [5]byte) bool {
	switch string(hdr[:]) {
	case "GET /", "HEAD ", "POST ", "PUT /", "OPTIO":
		return true
	}
	return false
}
