// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewTls(binding Address, config tls.Config) SessionfulTransport {
	return &TlsTransport{addr: binding.AsTCPAddr(), config: config}
}

func NewTlsTransport(binding Address, config tls.Config) TlsTransport {
	return TlsTransport{addr: binding.AsTCPAddr(), config: config}
}

type TlsTransport struct {
	config    tls.Config
	addr      net.TCPAddr
	conn      net.Listener
	receiveFn SessionfulConnectFunc
}

func (p *TlsTransport) IsZero() bool {
	return p.conn == nil && p.addr.IP == nil
}

func (p *TlsTransport) Listen(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
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
		// mimics tls.Listen
		return nil, errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
	}

	conn, err := net.ListenTCP("tcp", &p.addr)
	if err != nil {
		return nil, err
	}
	p.conn = tls.NewListener(conn, &p.config)
	p.receiveFn = receiveFn
	go runTcpListener(p.conn, receiveFn)
	return p, nil
}

func (p *TlsTransport) Outgoing(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	return &TlsTransport{p.config, p.addr, nil, receiveFn}, nil
}

func (p *TlsTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	return p.conn.Close()
}

func (p *TlsTransport) ConnectTo(to Address) (OutTransport, error) {
	return p.ConnectToExt(to, nil)
}

func (p *TlsTransport) ConnectToExt(to Address, peerVerify VerifyPeerCertificateFunc) (OutTransport, error) {
	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver)
	if err != nil {
		return nil, err
	}

	peerConfig := &p.config
	if peerVerify != nil {
		cfg := p.config
		cfg.VerifyPeerCertificate = peerVerify
		peerConfig = &cfg
	}

	var conn *tls.Conn
	if conn, err = tls.DialWithDialer(&net.Dialer{LocalAddr: &p.addr}, "tcp", to.String(), peerConfig); err != nil {
		return nil, err
	}

	if p.receiveFn != nil && !p.receiveFn(tcpAddr(&p.addr), to, conn, nil, nil) {
		_ = conn.Close()
		if p.conn != nil {
			_ = p.conn.Close()
		}
		return nil, errors.New("aborted")
	}
	return &tcpOutTransport{conn, nil, 0}, nil
}
