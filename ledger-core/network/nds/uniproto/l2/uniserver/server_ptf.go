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

	tcpProvider   l1.SessionfulTransportProvider
	tcpIncomingFn l1.SessionfulConnectFunc
	tcpOutgoingFn l1.SessionfulConnectFunc

	udpProvider   l1.SessionlessTransportProvider
	udpIncomingFn l1.SessionlessReceiveFunc

	tcpListen l1.OutTransportFactory
	udpListen l1.OutTransportFactory

	updateLocalAddr func(nwapi.Address)
}

func (p *peerTransportFactory) MaxSessionlessSize() uint16 {
	return p.udpProvider.MaxByteSize()
}

func (p *peerTransportFactory) SetSessionless(udp l1.SessionlessTransportProvider, slFn l1.SessionlessReceiveFunc) {
	switch {
	case udp == nil:
		panic(throw.IllegalValue())
	case slFn == nil:
		panic(throw.IllegalValue())
	case p.udpProvider != nil:
		panic(throw.IllegalState())
	}
	p.udpProvider = udp
	p.udpIncomingFn = slFn
}

func (p *peerTransportFactory) SetSessionful(tcp l1.SessionfulTransportProvider, outFn, sfFn l1.SessionfulConnectFunc) {
	switch {
	case tcp == nil:
		panic(throw.IllegalValue())
	case sfFn == nil:
		panic(throw.IllegalValue())
	case p.tcpProvider != nil:
		panic(throw.IllegalState())
	}
	p.tcpProvider = tcp
	p.tcpIncomingFn = sfFn
	p.tcpOutgoingFn = outFn
}

func (p *peerTransportFactory) HasTransports() bool {
	return p.udpProvider != nil && p.tcpProvider != nil
}

func (p *peerTransportFactory) Listen() (err error) {
	switch {
	case p.udpProvider == nil:
		panic(throw.IllegalState())
	case p.tcpProvider == nil:
		panic(throw.IllegalState())
	}

	if p.listen.DoStart(func() {
		var localAddr nwapi.Address
		if p.tcpListen, localAddr, err = p.tcpProvider.CreateListeningFactory(p.tcpIncomingFn); err != nil {
			return
		}

		if p.updateLocalAddr == nil {
			p.udpListen, _, err = p.udpProvider.CreateListeningFactory(p.udpIncomingFn)
			return
		}

		p.updateLocalAddr(localAddr)

		if p.udpListen, err = p.udpProvider.CreateListeningFactoryWithAddress(p.udpIncomingFn, localAddr); err != nil {
			return
		}
	}) {
		return
	}
	return throw.IllegalState()
}

func (p *peerTransportFactory) closeNotListen(err error) error {
	err = iokit.SafeCloseChain(p.tcpProvider, err)
	err = iokit.SafeCloseChain(p.udpProvider, err)
	return err
}

func (p *peerTransportFactory) Close() (err error) {
	if p.listen.DoDiscard(func() {
		err = p.closeNotListen(nil)
	}, func() {
		err = iokit.SafeCloseChain(p.tcpListen, err)
		err = iokit.SafeCloseChain(p.udpListen, err)
		err = p.closeNotListen(err)
	}) {
		return
	}
	return throw.IllegalState()
}

// LOCK: WARNING! This method is called under PeerTransport.mutex
func (p *peerTransportFactory) SessionlessConnectTo(to nwapi.Address) (l1.OneWayTransport, error) {
	if p.listen.IsActive() {
		return p.udpListen.ConnectTo(to)
	}

	// this is a bit inefficient, but ok
	out, err := p.udpProvider.CreateOutgoingOnlyFactory()
	if err != nil {
		return nil, err
	}
	return out.ConnectTo(to)
}

// LOCK: WARNING! This method is called under PeerTransport.mutex
func (p *peerTransportFactory) SessionfulConnectTo(to nwapi.Address) (l1.OneWayTransport, error) {
	if p.listen.IsActive() {
		return p.tcpListen.ConnectTo(to)
	}

	// this is a bit inefficient, but ok
	out, err := p.tcpProvider.CreateOutgoingOnlyFactory(p.tcpOutgoingFn)
	if err != nil {
		return nil, err
	}
	return out.ConnectTo(to)
}

func (p *peerTransportFactory) IsActive() bool {
	return !p.listen.WasStopped()
}
