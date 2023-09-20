package uniserver

import (
	"sync"

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

	// mutex protects outgoing factories
	mutex sync.RWMutex
	tcpOutgoing l1.OutTransportFactory
	maxSessionlessSize uint16
}

func (p *peerTransportFactory) MaxSessionlessSize() uint16 {
	return p.maxSessionlessSize // udpProvider.MaxByteSize()
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

func (p *peerTransportFactory) Listen() error {
	switch {
	case p.udpProvider == nil:
		panic(throw.IllegalState())
	case p.tcpProvider == nil:
		panic(throw.IllegalState())
	case p.listen.IsActive():
		panic(throw.IllegalState())
	}

	if max := p.udpProvider.MaxByteSize(); max < p.maxSessionlessSize {
		p.maxSessionlessSize = max
	}

	tcpListen, localAddr, err := p.tcpProvider.CreateListeningFactory(p.tcpIncomingFn)
	if err != nil {
		return err
	}
	var udpListen l1.OutTransportFactory

	defer func() {
		_ = iokit.SafeClose(tcpListen)
		_ = iokit.SafeClose(udpListen)
	}()

	if p.updateLocalAddr == nil {
		udpListen, _, err = p.udpProvider.CreateListeningFactory(p.udpIncomingFn)
	} else {
		udpListen, err = p.udpProvider.CreateListeningFactoryWithAddress(p.udpIncomingFn, localAddr)
	}

	if err != nil {
		return err
	}
	if !p.listen.DoStart(func() {
		if p.updateLocalAddr != nil {
			p.updateLocalAddr(localAddr)
		}

		p.tcpListen = tcpListen
		p.udpListen = udpListen
		tcpListen, udpListen = nil, nil
	}) {
		panic(throw.IllegalState())
	}
	return nil
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
	if p.maxSessionlessSize == 0 {
		return nil, throw.E("sessionless is disabled")
	}

	if p.listen.IsActive() {
		return p.udpListen.ConnectTo(to)
	}

	return nil, throw.E("sessionless is unavailable without listener")
}

// LOCK: WARNING! This method is called under PeerTransport.mutex
func (p *peerTransportFactory) SessionfulConnectTo(to nwapi.Address) (l1.OneWayTransport, error) {
	if p.listen.IsActive() {
		return p.tcpListen.ConnectTo(to)
	}

	t, err := p.getOutgoing()
	if err != nil {
		return nil, err
	}
	return t.ConnectTo(to)
}

func (p *peerTransportFactory) IsSessionlessAllowed(size int) bool {
	switch {
	case size < 0:
	case p.maxSessionlessSize == 0:
	case size > int(p.maxSessionlessSize):
	default:
		return p.listen.WasStarted()
	}
	return false
}

func (p *peerTransportFactory) IsActive() bool {
	return !p.listen.WasStopped()
}

func (p *peerTransportFactory) getOutgoing() (l1.OutTransportFactory, error) {
	p.mutex.RLock()
	if f := p.tcpOutgoing; f != nil {
		p.mutex.RUnlock()
		return f, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if f := p.tcpOutgoing; f != nil {
		return f, nil
	}

	t, err := p.tcpProvider.CreateOutgoingOnlyFactory(p.tcpOutgoingFn)
	if err != nil {
		return nil, err
	}
	p.tcpOutgoing = t
	return t, err
}
