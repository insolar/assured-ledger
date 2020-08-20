// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"context"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/ratelimiter"
)

func NewTCP(binding nwapi.Address) SessionfulTransportProvider {
	return &TCPTransport{addr: binding.AsTCPAddr()}
}

func NewTCPTransport(binding nwapi.Address) TCPTransport {
	return TCPTransport{addr: binding.AsTCPAddr()}
}

type TCPTransport struct {
	addr      net.TCPAddr
	conn      *net.TCPListener
	receiveFn SessionfulConnectFunc
}

func (p *TCPTransport) IsZero() bool {
	return p.conn == nil && p.addr.IP == nil
}

func (p *TCPTransport) CreateListeningFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
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
	go runTCPListener(p.conn, receiveFn)
	return p, nil
}

func (p *TCPTransport) CreateOutgoingOnlyFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	return &TCPTransport{p.addr, nil, receiveFn}, nil
}

func (p *TCPTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	return p.conn.Close()
}

func (p *TCPTransport) LocalAddr() nwapi.Address {
	if p.conn != nil {
		return nwapi.AsAddress(p.conn.Addr())
	}
	return nwapi.AsAddress(&p.addr)
}

func (p *TCPTransport) ConnectTo(to nwapi.Address, preference nwapi.Preference) (OutTransport, error) {
	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver, preference)
	if err != nil {
		return nil, err
	}

	remote := to.AsTCPAddr()
	local := p.addr
	local.Port = 0

	var conn *net.TCPConn
	if conn, err = net.DialTCP("tcp", &local, &remote); err != nil {
		return nil, err
	}

	tcpOut := tcpOutTransport{conn, nil, 0}
	if p.receiveFn == nil {
		return &tcpSemiTransport{tcpOut, func(_, _ nwapi.Address, conn io.ReadWriteCloser, _ OutTransport, _ error) bool {
			_ = conn.(*net.TCPConn).CloseRead()
			return false
		}}, nil
	}
	return &tcpSemiTransport{tcpOut, p.receiveFn}, nil
}

func runTCPListener(listenConn net.Listener, receiveFn SessionfulConnectFunc) {
	defer func() {
		_ = listenConn.Close()
		_ = recover()
	}()

	local := nwapi.FromTCPAddr(listenConn.Addr().(*net.TCPAddr))

	for {
		conn, err := listenConn.Accept()
		switch {
		case err == nil:
			w := &tcpOutTransport{conn, nil, 0}
			if !receiveFn(local, nwapi.FromTCPAddr(conn.RemoteAddr().(*net.TCPAddr)), conn, w, nil) {
				break
			}
			continue
		case err == io.EOF:
			return
		case !receiveFn(local, nwapi.Address{}, nil, nil, err):
			return
		}
		if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
			return
		}
	}
}

var _ TwoWayTransport = &tcpOutTransport{}
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

func (p *tcpOutTransport) TwoWayConn() io.ReadWriteCloser {
	return p.conn
}

func (p *tcpOutTransport) NetConn() net.Conn {
	return p.conn
}

var _ SemiTransport = &tcpSemiTransport{}

type tcpSemiTransport struct {
	tcpOutTransport
	receiveFn SessionfulConnectFunc
}

func (p *tcpSemiTransport) ConnectReceiver(fn SessionfulConnectFunc) (bool, TwoWayTransport) {
	if p.receiveFn == nil {
		return false, nil
	}
	if fn == nil {
		fn = p.receiveFn
	}
	p.receiveFn = nil
	return fn(
			nwapi.AsAddress(p.conn.LocalAddr()),
			nwapi.AsAddress(p.conn.RemoteAddr()),
			p.conn, nil, nil),
		&p.tcpOutTransport
}
