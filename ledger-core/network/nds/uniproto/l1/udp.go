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

func NewUDP(binding nwapi.Address, preference nwapi.Preference, maxByteSize uint16) SessionlessTransportProvider {
	if maxByteSize == 0 {
		panic(throw.IllegalValue())
	}
	return udpProvider{addr: binding.AsUDPAddr(), preference: preference, maxByteSize: maxByteSize}
}

type udpProvider struct {
	addr        net.UDPAddr
	preference  nwapi.Preference
	maxByteSize uint16
}

func (v udpProvider) IsZero() bool {
	return v.addr.IP == nil
}

func (v udpProvider) MaxByteSize() uint16 {
	return v.maxByteSize
}

func (v udpProvider) CreateListeningFactoryWithAddress(receiveFn SessionlessReceiveFunc, binding nwapi.Address) (OutTransportFactory, error) {
	switch {
	case binding.IsZero():
		panic(throw.IllegalValue())
	case !binding.HasPort():
		panic(throw.IllegalValue())
	}

	// this instance is by-value, so it is safe to change it here
	v.addr = binding.AsUDPAddr()
	out, _, err := v.CreateListeningFactory(receiveFn)
	return out, err
}

// SessionlessReceiveFunc MUST NOT reuse (b) after return
func (v udpProvider) CreateListeningFactory(receiveFn SessionlessReceiveFunc) (OutTransportFactory, nwapi.Address, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case v.IsZero():
		panic(throw.IllegalState())
	}

	t := &udpTransportFactory{v.addr, nil, v.preference, v.maxByteSize}

	var err error
	t.conn, err = net.ListenUDP("udp", &t.addr)
	if err != nil {
		return nil, nwapi.Address{}, err
	}
	listenAddr := nwapi.FromUDPAddr(t.conn.LocalAddr().(*net.UDPAddr))

	go runUDPListener(t, receiveFn)
	return t, listenAddr, nil
}

func (v udpProvider) CreateOutgoingOnlyFactory() (OutTransportFactory, error) {
	if v.IsZero() {
		panic(throw.IllegalState())
	}

	t := &udpTransportFactory{v.addr, nil, v.preference, v.maxByteSize}

	var err error
	t.conn, err = net.DialUDP("udp", &t.addr, nil)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (v udpProvider) Close() error {
	return nil
}

/***********************/

type udpTransportFactory struct {
	addr        net.UDPAddr // this field is passed by udpProvider as a pointer into Dial
	conn        *net.UDPConn
	preference  nwapi.Preference
	maxByteSize uint16
}

func (p *udpTransportFactory) IsZero() bool {
	return p.addr.IP == nil
}

func (p *udpTransportFactory) MaxByteSize() uint16 {
	return p.maxByteSize
}

func (p *udpTransportFactory) ConnectTo(to nwapi.Address) (OneWayTransport, error) {
	if p.IsZero() {
		return nil, throw.IllegalState()
	}

	if !to.IsNetCompatible() {
		return nil, nil
	}

	resolved, err := to.Resolve(context.Background(), net.DefaultResolver, p.preference)
	if err != nil {
		return nil, err
	}
	return &udpOutTransport{resolved.AsUDPAddr(), nil, p.conn, p.maxByteSize, 0 }, nil
}

func (p *udpTransportFactory) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func runUDPListener(t *udpTransportFactory, receiveFn SessionlessReceiveFunc) {
	defer func() {
		_ = t.conn.Close()
		_ = recover()
	}()

	to := nwapi.FromUDPAddr(t.conn.LocalAddr().(*net.UDPAddr))
	buf := make([]byte, t.maxByteSize)

	for {
		n, addr, err := t.conn.ReadFromUDP(buf)

		if !receiveFn(to, nwapi.FromUDPAddr(addr), buf[:n], err) {
			break
		}
		if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
			break
		}
	}
}

var _ OneWayTransport = &udpOutTransport{}

type udpOutTransport struct {
	addr  net.UDPAddr
	quota ratelimiter.RateQuota
	conn  *net.UDPConn
	maxByteSize uint16
	tag   int
}

var errTooLarge = errors.New("exceeds UDP limit")

func (p *udpOutTransport) Write(b []byte) (int, error) {
	switch {
	case p.conn == nil:
		return 0, throw.IllegalState()
	case len(b) == 0:
		return 0, nil
	case len(b) > int(p.maxByteSize):
		return 0, errTooLarge
	}

	// UDP can only be sent in one chunk,  so we have to accumulate quota
	if err := iokit.RateLimitedBySize(int64(len(b)), p.quota); err != nil {
		return 0, err
	}

	return p.conn.WriteToUDP(b, &p.addr)
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
	case n > int64(p.maxByteSize):
		n = int64(p.maxByteSize) + 1 // to allow too large error
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
	p.conn = nil
	return nil
}

func (p *udpOutTransport) GetTag() int {
	return p.tag
}

func (p *udpOutTransport) SetTag(tag int) {
	p.tag = tag
}

func (p *udpOutTransport) WithQuota(q ratelimiter.RateQuota) OneWayTransport {
	if p.quota == q {
		return p
	}
	cp := *p
	cp.quota = q
	return &cp
}
