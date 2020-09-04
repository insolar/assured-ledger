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

func NewTCP(binding nwapi.Address, preference nwapi.Preference) SessionfulTransportProvider {
	return tcpProvider{addr: binding.AsTCPAddr(), preference: preference}
}

type tcpProvider struct {
	addr       net.TCPAddr
	preference nwapi.Preference
}

func (v tcpProvider) IsZero() bool {
	return v.addr.IP == nil
}

func (v tcpProvider) CreateListeningFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, nwapi.Address, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case v.IsZero():
		panic(throw.IllegalState())
	}

	listener, err := ListenTCPWithReuse("tcp", &v.addr)
	if err != nil {
		return nil, nwapi.Address{}, err
	}

	localAddr := *listener.Addr().(*net.TCPAddr)

	t := &tcpTransportFactory{listener, receiveFn, localAddr, v.preference }

	go runTCPListener(listener, receiveFn)
	return t, nwapi.FromTCPAddr(&localAddr), nil
}

func (v tcpProvider) CreateOutgoingOnlyFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	t := &tcpTransportFactory{nil, receiveFn, v.addr, v.preference }
	t.addr.Port = 0

	return t, nil
}

func (v tcpProvider) Close() error {
	return nil
}

/*********************************/

type tcpTransportFactory struct {
	listener   *net.TCPListener
	receiveFn  SessionfulConnectFunc
	addr       net.TCPAddr
	preference nwapi.Preference
}

func (p *tcpTransportFactory) IsZero() bool {
	return p.addr.IP == nil
}

func (p *tcpTransportFactory) Close() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

func (p *tcpTransportFactory) ConnectTo(to nwapi.Address) (OneWayTransport, error) {
	if !to.IsNetCompatible() {
		return nil, nil
	}

	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver, p.preference)
	if err != nil {
		return nil, err
	}

	remote := to.AsTCPAddr()

	var conn *net.TCPConn

	if p.addr.Port == 0 {
		if conn, err = net.DialTCP("tcp", &p.addr, &remote); err != nil {
			return nil, err
		}
	} else {
		if conn, err = DialTCPWithReuse("tcp", &p.addr, &remote); err != nil {
			return nil, err
		}
	}

	tcpOut := tcpOutTransport{conn, nil, 0}
	if p.receiveFn == nil {
		// by default - when ConnectReceiver(nil) is invoked
		// then the unused read-side of TCP will be closed as a precaution
		return &tcpSemiTransport{tcpOut, func(_, _ nwapi.Address, conn io.ReadWriteCloser, _ OneWayTransport, _ error) bool {
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

func (p *tcpOutTransport) WithQuota(q ratelimiter.RateQuota) OneWayTransport {
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
	ok := fn(
		nwapi.AsAddress(p.conn.LocalAddr()),
		nwapi.AsAddress(p.conn.RemoteAddr()),
		p.conn, nil, nil)

	return ok, &p.tcpOutTransport
}
