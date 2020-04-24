// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ PeerTransportFactory = &peerTransportFactory{}

type peerTransportFactory struct {
	listen atomickit.StartStopFlag

	udp        l1.SessionlessTransport
	udpReceive l1.SessionlessReceiveFunc

	tcp        l1.SessionfulTransport
	tcpConnect l1.SessionfulConnectFunc

	udpListen l1.OutTransportFactory
	tcpListen l1.OutTransportFactory
}

func (p *peerTransportFactory) MaxSessionlessSize() uint16 {
	return p.udp.MaxByteSize()
}

func (p *peerTransportFactory) SetSessionless(udp l1.SessionlessTransport, slFn l1.SessionlessReceiveFunc) {
	switch {
	case udp == nil:
		panic(throw.IllegalValue())
	case slFn == nil:
		panic(throw.IllegalValue())
	case p.udp != nil:
		panic(throw.IllegalState())
	}
	p.udp = udp
	p.udpReceive = slFn
}

func (p *peerTransportFactory) SetSessionful(tcp l1.SessionfulTransport, sfFn l1.SessionfulConnectFunc) {
	switch {
	case tcp == nil:
		panic(throw.IllegalValue())
	case sfFn == nil:
		panic(throw.IllegalValue())
	case p.tcp != nil:
		panic(throw.IllegalState())
	}
	p.tcp = tcp
	p.tcpConnect = sfFn
}

func (p *peerTransportFactory) HasTransports() bool {
	return p.udp != nil && p.tcp != nil
}

func (p *peerTransportFactory) Listen() (err error) {
	switch {
	case p.udp == nil:
		panic(throw.IllegalState())
	case p.tcp == nil:
		panic(throw.IllegalState())
	}

	if p.listen.DoStart(func() {
		if p.udpListen, err = p.udp.Listen(p.udpReceive); err != nil {
			return
		}
		if p.tcpListen, err = p.tcp.Listen(p.tcpConnect); err != nil {
			return
		}
	}) {
		return
	}
	return throw.IllegalState()
}

func (p *peerTransportFactory) Close() (err error) {
	if p.listen.DoDiscard(func() {
		err = iokit.SafeCloseChain(p.tcp, err)
		err = iokit.SafeCloseChain(p.udp, err)
	}, func() {
		err = iokit.SafeCloseChain(p.tcpListen, err)
		err = iokit.SafeCloseChain(p.udpListen, err)
		err = iokit.SafeCloseChain(p.tcp, err)
		err = iokit.SafeCloseChain(p.udp, err)
	}) {
		return
	}
	return throw.IllegalState()
}

// LOCK: WARNING! This method is called under PeerTransport.mutex
func (p *peerTransportFactory) SessionlessConnectTo(to nwapi.Address) (l1.OutTransport, error) {
	if p.listen.IsActive() {
		return p.udpListen.ConnectTo(to)
	}
	if out, err := p.udp.Outgoing(); err != nil {
		return nil, err
	} else {
		return out.ConnectTo(to)
	}
}

// LOCK: WARNING! This method is called under PeerTransport.mutex
func (p *peerTransportFactory) SessionfulConnectTo(to nwapi.Address) (l1.OutTransport, error) {
	if p.listen.IsActive() {
		return p.tcpListen.ConnectTo(to)
	}
	if out, err := p.tcp.Outgoing(p.tcpConnect); err != nil {
		return nil, err
	} else {
		return out.ConnectTo(to)
	}
}

func (p *peerTransportFactory) IsActive() bool {
	return !p.listen.WasStopped()
}
