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

func NewTLS(binding nwapi.Address, preference nwapi.Preference, config *tls.Config) SessionfulTransportProvider {
	return TLSProvider{addr: binding.AsTCPAddr(), preference: preference, config: config}
}

type TLSProvider struct {
	addr       net.TCPAddr
	config     *tls.Config
	preference nwapi.Preference
}

func (v TLSProvider) IsZero() bool {
	return v.addr.IP == nil
}

func (v TLSProvider) CreateListeningFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, nwapi.Address, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case v.IsZero():
		panic(throw.IllegalState())
	case len(v.config.Certificates) > 0 || v.config.GetCertificate != nil || v.config.GetConfigForClient != nil:
		// ok
	default:
		// mimics tls.CreateListeningFactory
		return nil, nwapi.Address{}, errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
	}

	tcpConn, err := net.ListenTCP("tcp", &v.addr)
	if err != nil {
		return nil, nwapi.Address{}, err
	}

	localAddr := *tcpConn.Addr().(*net.TCPAddr)

	tlsConn := tls.NewListener(tcpConn, v.config)
	t := &TLSTransport{tlsConn, receiveFn, v.config, localAddr, v.preference}
	t.addr.Port = 0

	go runTCPListener(tlsConn, t.tlsConnect)
	return t, nwapi.FromTCPAddr(&localAddr), nil
}

func (v TLSProvider) CreateOutgoingOnlyFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	return &TLSTransport{nil, receiveFn, v.config, v.addr, v.preference}, nil
}

func (v TLSProvider) Close() error {
	return nil
}

/*********************************/

type TLSTransport struct {
	conn      net.Listener
	receiveFn SessionfulConnectFunc
	config    *tls.Config
	addr      net.TCPAddr
	preference nwapi.Preference
}

func (p *TLSTransport) IsZero() bool {
	return p.addr.IP == nil
}

func (p *TLSTransport) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *TLSTransport) LocalAddr() nwapi.Address {
	if p.conn != nil {
		return nwapi.AsAddress(p.conn.Addr())
	}
	return nwapi.AsAddress(&p.addr)
}

func (p *TLSTransport) ConnectTo(to nwapi.Address) (OutTransport, error) {
	return p.ConnectToExt(to, nil)
}

func (p *TLSTransport) ConnectToExt(to nwapi.Address, peerVerify VerifyPeerCertificateFunc) (OutTransport, error) {
	if !to.IsNetCompatible() {
		return nil, nil
	}

	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver, p.preference)
	if err != nil {
		return nil, err
	}

	peerConfig := p.config
	if peerVerify != nil {
		cfg := p.config.Clone()
		cfg.VerifyPeerCertificate = peerVerify
		peerConfig = cfg
	}

	var conn *tls.Conn
	if conn, err = tls.DialWithDialer(&net.Dialer{LocalAddr: &p.addr}, "tcp", to.String(), peerConfig); err != nil {
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

	// err = throw.WithDetails(err, l2.ConnErrDetails{Local:local, Remote:remote})
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
