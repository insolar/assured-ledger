// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

var _ PeerTransportFactory = &peerTransportFactory{}

type peerTransportFactory struct {
	listen atomickit.StartStopFlag

	tcp         l1.SessionfulTransportProvider
	tcpConnect  l1.SessionfulConnectFunc
	tcpOutgoing l1.SessionfulConnectFunc

	udp        l1.SessionlessTransportProvider
	udpReceive l1.SessionlessReceiveFunc

	tcpListen l1.OutTransportFactory
	udpListen l1.OutTransportFactory

	updateLocalAddr func(nwapi.Address)

	preference nwapi.Preference
}

func (p *peerTransportFactory) MaxSessionlessSize() uint16 {
	return p.udp.MaxByteSize()
}

func (p *peerTransportFactory) SetSessionless(udp l1.SessionlessTransportProvider, slFn l1.SessionlessReceiveFunc) {
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

func (p *peerTransportFactory) SetSessionful(tcp l1.SessionfulTransportProvider, outFn, sfFn l1.SessionfulConnectFunc) {
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
	p.tcpOutgoing = outFn
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
		if p.tcpListen, err = p.tcp.CreateListeningFactory(p.tcpConnect); err != nil {
			return
		}
		if p.updateLocalAddr == nil {
			p.udpListen, err = p.udp.CreateListeningFactory(p.udpReceive)
			return
		}

		localAddr := p.tcpListen.LocalAddr()
		p.updateLocalAddr(localAddr)

		if p.udpListen, err = p.udp.CreateListeningFactoryWithAddress(p.udpReceive, localAddr); err != nil {
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
	switch {
	case !to.IsNetCompatible():
		return nil, nil
	case p.listen.IsActive():
		return p.udpListen.ConnectTo(to, p.preference)
	}
	out, err := p.udp.CreateOutgoingOnlyFactory()
	if err != nil {
		return nil, err
	}
	return out.ConnectTo(to, p.preference)
}

// LOCK: WARNING! This method is called under PeerTransport.mutex
func (p *peerTransportFactory) SessionfulConnectTo(to nwapi.Address) (l1.OutTransport, error) {
	switch {
	case !to.IsNetCompatible():
		return nil, nil
	case p.listen.IsActive():
		return p.tcpListen.ConnectTo(to, p.preference)
	}
	out, err := p.tcp.CreateOutgoingOnlyFactory(p.tcpOutgoing)
	if err != nil {
		return nil, err
	}
	return out.ConnectTo(to, p.preference)
}

func (p *peerTransportFactory) IsActive() bool {
	return !p.listen.WasStopped()
}
