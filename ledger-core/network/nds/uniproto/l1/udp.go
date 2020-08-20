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

	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/ratelimiter"
)

// const MinUdpSize = 1300
const MaxUDPSize = 2048

func NewUDP(binding nwapi.Address, maxByteSize uint16) SessionlessTransportProvider {
	if maxByteSize == 0 {
		panic(throw.IllegalValue())
	}
	return &UDPTransport{addr: binding.AsUDPAddr(), maxByteSize: maxByteSize}
}

func NewUDPTransport(binding nwapi.Address, maxByteSize uint16) UDPTransport {
	if maxByteSize == 0 {
		panic(throw.IllegalValue())
	}
	return UDPTransport{addr: binding.AsUDPAddr(), maxByteSize: maxByteSize}
}

type UDPTransport struct {
	addr        net.UDPAddr
	conn        *net.UDPConn
	maxByteSize uint16
}

func (p *UDPTransport) IsZero() bool {
	return p.conn == nil && p.addr.IP == nil
}

// SessionlessReceiveFunc MUST NOT reuse (b) after return
func (p *UDPTransport) CreateListeningFactoryWithAddress(receiveFn SessionlessReceiveFunc, binding nwapi.Address) (OutTransportFactory, error) {
	if binding.IsZero() {
		return p.CreateListeningFactory(receiveFn)
	}
	cp := &UDPTransport{binding.AsUDPAddr(), nil, p.maxByteSize}
	return cp.CreateListeningFactory(receiveFn)
}

// SessionlessReceiveFunc MUST NOT reuse (b) after return
func (p *UDPTransport) CreateListeningFactory(receiveFn SessionlessReceiveFunc) (OutTransportFactory, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case p.conn != nil:
		return nil, throw.IllegalState()
	case p.addr.IP == nil:
		return nil, throw.IllegalState()
	}
	var err error
	p.conn, err = net.ListenUDP("udp", &p.addr)
	if err != nil {
		return nil, err
	}
	go p.run(receiveFn)
	return p, nil
}

func (p *UDPTransport) CreateOutgoingOnlyFactory() (OutTransportFactory, error) {
	switch {
	case p.conn != nil:
		return p, nil
	case p.addr.IP == nil:
		return nil, throw.IllegalState()
	}
	cp := &UDPTransport{p.addr, nil, p.maxByteSize}
	var err error
	cp.conn, err = net.DialUDP("udp", &cp.addr, nil)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

func (p *UDPTransport) MaxByteSize() uint16 {
	return p.maxByteSize
}

func (p *UDPTransport) LocalAddr() nwapi.Address {
	if p.conn != nil {
		return nwapi.AsAddress(p.conn.LocalAddr())
	}
	return nwapi.AsAddress(&p.addr)
}

func (p *UDPTransport) ConnectTo(to nwapi.Address, preference nwapi.Preference) (OutTransport, error) {
	if p.conn == nil {
		return nil, throw.IllegalState()
	}

	resolved, err := to.Resolve(context.Background(), net.DefaultResolver, preference)
	if err != nil {
		return nil, err
	}
	return &udpOutTransport{resolved.AsUDPAddr(), nil, p, 0}, nil
}

func (p *UDPTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	return p.conn.Close()
}

func (p *UDPTransport) run(receiveFn SessionlessReceiveFunc) {
	if p.conn == nil {
		panic(throw.IllegalState())
	}

	defer func() {
		_ = p.conn.Close()
		_ = recover()
	}()

	to := nwapi.FromUDPAddr(p.conn.LocalAddr().(*net.UDPAddr))
	buf := make([]byte, p.maxByteSize)

	for {
		n, addr, err := p.conn.ReadFromUDP(buf)

		if !receiveFn(to, nwapi.FromUDPAddr(addr), buf[:n], err) {
			break
		}
		if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
			break
		}
	}
}

var _ OutTransport = &udpOutTransport{}

type udpOutTransport struct {
	addr  net.UDPAddr
	quota ratelimiter.RateQuota
	conn  *UDPTransport
	tag   int
}

var errTooLarge = errors.New("exceeds UDP limit")

func (p *udpOutTransport) Write(b []byte) (int, error) {
	switch {
	case p.conn == nil:
		return 0, throw.IllegalState()
	case len(b) == 0:
		return 0, nil
	case len(b) > int(p.conn.maxByteSize):
		return 0, errTooLarge
	}

	// UDP can only be sent in one chunk,  so we have to accumulate quota
	if err := iokit.RateLimitedBySize(int64(len(b)), p.quota); err != nil {
		return 0, err
	}

	return p.conn.conn.WriteToUDP(b, &p.addr)
}

func (p *udpOutTransport) SendBytes(payload []byte) error {
	_, err := p.Write(payload)
	return err
}

func (p *udpOutTransport) ReadFrom(r io.Reader) (int64, error) {
	n := iokit.LimitOfReader(r)
	switch {
	case n == 0:
		return 0, nil
	case n > int64(p.conn.maxByteSize):
		n = int64(p.conn.maxByteSize) + 1 // to allow too large error
	}

	b := make([]byte, n)
	total, err := io.ReadAtLeast(r, b, 0)
	if err2 := p.SendBytes(b[:total]); err2 != nil {
		return int64(total), err2
	}
	return int64(total), err
}

func (p *udpOutTransport) Send(payload io.WriterTo) error {
	_, err := payload.WriteTo(p)
	return err
}

func (p *udpOutTransport) Close() error {
	if p.conn == nil {
		return throw.IllegalState()
	}
	p.conn = nil
	return nil
}

func (p *udpOutTransport) GetTag() int {
	return p.tag
}

func (p *udpOutTransport) SetTag(tag int) {
	p.tag = tag
}

func (p *udpOutTransport) WithQuota(q ratelimiter.RateQuota) OutTransport {
	if p.quota == q {
		return p
	}
	cp := *p
	cp.quota = q
	return &cp
}
