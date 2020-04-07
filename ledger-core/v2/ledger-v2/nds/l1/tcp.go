// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewTcp(binding Address) SessionfulTransport {
	return &TcpTransport{addr: binding.AsTCPAddr()}
}

func NewTcpTransport(binding Address) TcpTransport {
	return TcpTransport{addr: binding.AsTCPAddr()}
}

type TcpTransport struct {
	addr      net.TCPAddr
	conn      *net.TCPListener
	receiveFn SessionfulConnectFunc
}

func (p *TcpTransport) IsZero() bool {
	return p.conn == nil && p.addr.IP == nil
}

func (p *TcpTransport) Listen(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case p.conn != nil:
		return nil, throw.IllegalState()
	case p.addr.IP == nil:
		return nil, throw.IllegalState()
	}
	var err error
	p.conn, err = net.ListenTCP("tcp", &p.addr)
	if err != nil {
		return nil, err
	}
	p.receiveFn = receiveFn
	go runTcpListener(p.conn, receiveFn)
	return p, nil
}

func (p *TcpTransport) Outgoing(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	return &TcpTransport{p.addr, nil, receiveFn}, nil
}

func (p *TcpTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	return p.conn.Close()
}

func (p *TcpTransport) ConnectTo(to Address) (OutTransport, error) {
	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver)
	if err != nil {
		return nil, err
	}

	remote := to.AsTCPAddr()
	var conn *net.TCPConn
	if conn, err = net.DialTCP("tcp", &p.addr, &remote); err != nil {
		return nil, err
	}

	switch {
	case p.receiveFn == nil:
		_ = conn.CloseRead()
	case !p.receiveFn(tcpAddr(&p.addr), to, conn, nil, nil):
		_ = conn.Close()
		if p.conn != nil {
			_ = p.conn.Close()
		}
		return nil, errors.New("aborted")
	}
	return &tcpOutTransport{conn, nil, 0}, nil
}

func runTcpListener(listenConn net.Listener, receiveFn SessionfulConnectFunc) {
	defer func() {
		_ = listenConn.Close()
		recover()
	}()

	local := tcpAddr(listenConn.Addr().(*net.TCPAddr))

	for {
		conn, err := listenConn.Accept()
		if err == nil {
			w := &tcpOutTransport{conn, nil, 0}
			if !receiveFn(local, tcpAddr(conn.RemoteAddr().(*net.TCPAddr)), conn, w, nil) {
				break
			}
			continue
		}
		if !receiveFn(local, Address{}, nil, nil, err) {
			break
		}
		if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
			break
		}
	}
}

var _ OutNetTransport = &tcpOutTransport{}

type tcpOutTransport struct {
	conn  net.Conn
	quota ratelimiter.RateQuota
	tag   int
}

func (p *tcpOutTransport) Write(b []byte) (n int, err error) {
	if p.conn == nil {
		return 0, throw.IllegalState()
	}
	return iokit.RateLimitedByteCopy(p.conn.Write, b, p.quota)
}

func (p *tcpOutTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	err := p.conn.Close()
	p.conn = nil
	return err
}

func (p *tcpOutTransport) ReadFrom(r io.Reader) (int64, error) {
	if p.conn == nil {
		return 0, throw.IllegalState()
	}
	return iokit.RateLimitedCopy(p.conn, r, p.quota)
}

func (p *tcpOutTransport) Send(payload io.WriterTo) error {
	_, err := payload.WriteTo(p)
	return err
}

func (p *tcpOutTransport) SendBytes(payload []byte) error {
	_, err := p.Write(payload)
	return err
}

func (p *tcpOutTransport) GetTag() int {
	return p.tag
}

func (p *tcpOutTransport) SetTag(tag int) {
	p.tag = tag
}

func (p *tcpOutTransport) WithQuota(q ratelimiter.RateQuota) OutTransport {
	if p.quota == q {
		return p
	}
	cp := *p
	cp.quota = q
	return &cp
}

func (p *tcpOutTransport) Conn() net.Conn {
	return p.conn
}
