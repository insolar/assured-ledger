// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"errors"
	"io"
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PeerTransportFactory interface {
	// LOCK: WARNING! This method is called under PeerTransport.mutex
	SessionlessConnectTo(to nwapi.Address) (l1.OutTransport, error)
	// LOCK: WARNING! This method is called under PeerTransport.mutex
	SessionfulConnectTo(to nwapi.Address) (l1.OutTransport, error)
	IsActive() bool
	MaxSessionlessSize() uint16
}

type PeerTransportCentral struct {
	factory PeerTransportFactory

	maxSessionlessSize uint16
	maxPeerConn        uint8
	preferHttp         bool
}

func (p *PeerTransportCentral) checkActive(deadPeer bool) error {
	if deadPeer || !p.factory.IsActive() {
		return errors.New("aborted")
	}
	return nil
}

func (p *PeerTransportCentral) getTransportStreamFormat(limitedLength bool, peerPreferHttp bool) TransportStreamFormat {
	switch {
	case !p.preferHttp && !peerPreferHttp:
		if limitedLength {
			return BinaryLimitedLength
		}
		return BinaryUnlimitedLength
	case limitedLength:
		return HttpLimitedLength
	default:
		return HttpUnlimitedLength
	}
}

var _ uniproto.OutTransport = &PeerTransport{}

var peerUID atomickit.Uint64

func NextPeerUID() uint64 {
	n := peerUID.Add(1)
	if n == 0 {
		panic(throw.IllegalState())
	}
	return n
}

type PeerTransport struct {
	central *PeerTransportCentral
	dead    atomickit.OnceFlag
	uid     uint64

	aliases []nwapi.Address

	rateQuota ratelimiter.RWRateQuota

	// LOCK: WARNING! This mutex can be acquired under PeerManager.peerMutex
	mutex      sync.RWMutex
	smallMutex sync.Mutex
	largeMutex sync.Mutex

	addrIndex uint8
	connCount uint8

	sessionless l1.OutTransport
	sessionful  l1.OutTransport
	connections []io.Closer
}

func (p *PeerTransport) kill() []nwapi.Address {
	if !p.dead.Set() {
		return nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = iokit.SafeClose(p.sessionless)
	ss := p.connections

	p.sessionless = nil
	p.sessionful = nil
	p.connections = nil

	for _, s := range ss {
		_ = s.Close()
	}
	aliases := p.aliases
	p.aliases = nil
	return aliases
}

func (p *PeerTransport) checkActive() error {
	return p.central.checkActive(p.dead.IsSet())
}

func (p *PeerTransport) resetTransport(t l1.OutTransport, discardCurrentAddr bool) (ok bool, hasMore bool) {
	return p.resetConnection(t, discardCurrentAddr)
}

func (p *PeerTransport) resetConnection(t io.Closer, discardCurrentAddr bool) (ok bool, hasMore bool) {
	if t == nil {
		return false, false
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	ok, hasMore, _ = p._resetConnection(t, discardCurrentAddr)
	return
}

func (p *PeerTransport) _resetConnection(t io.Closer, discardCurrentAddr bool) (ok bool, hasMore bool, err error) {
	if t == p.sessionless {
		p.sessionless = nil
		return true, false, t.Close()
	}

	if t == p.sessionful {
		p.sessionful = nil
	}

	for i, s := range p.connections {
		if s != t {
			if nout, ok := s.(l1.TwoWayTransport); ok && nout.TwoWayConn() == t {
				err = t.Close()
				_ = s.Close()
			} else {
				continue
			}
		} else {
			err = s.Close()
		}

		if n := len(p.connections); n <= 1 {
			p.connections = nil
		} else {
			copy(p.connections[i:], p.connections[i+1:])
			p.connections = p.connections[:n-1]
		}

		hasMore = true
		if discardCurrentAddr {
			p.addrIndex++
			if int(p.addrIndex) >= len(p.aliases) {
				p.addrIndex = 0
				hasMore = false
			}
		}
		return true, hasMore, err
	}
	return false, false, nil
}

type connectFunc = func(to nwapi.Address) (l1.OutTransport, error)

func (p *PeerTransport) tryConnect(factoryFn connectFunc, startIndex uint8, limit TransportStreamFormat) (uint8, l1.OutTransport, error) {

	for index := startIndex; ; {
		addr := p.aliases[index]

		var err error
		switch addr.AddrNetwork() {
		case nwapi.DNS:
			// primary address is always resolved
			if index == 0 {
				continue
			}
			fallthrough
		case nwapi.IP:
			var t l1.OutTransport
			t, err = factoryFn(addr)

			if err := p.checkActive(); err != nil {
				return index, nil, err
			}

			switch {
			case err != nil:
				break
			case p.rateQuota != nil:
				t = t.WithQuota(p.rateQuota.WriteBucket())
				fallthrough
			default:
				t.SetTag(int(limit))
				return index, t, nil
			}
		case nwapi.HostPK:
			// PK is addressable, but is not connectable
			continue
		default:
			err = throw.NotImplemented() // TODO redirected addresses
		}

		if s := p.sessionless; s != nil {
			// sessionless connection must follow sessionful
			// otherwise it is not possible to detect a non-passable connection
			p.sessionless = nil
			_ = s.Close()
		}

		if index++; int(index) >= len(p.aliases) {
			index = 0
		}
		if index == startIndex {
			return index, nil, err
		}
	}
}

func (p *PeerTransport) getSessionlessTransport() (l1.OutTransport, error) {
	p.mutex.RLock()
	if t := p.sessionless; t != nil {
		p.mutex.RUnlock()
		return t, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.sessionless != nil {
		return p.sessionless, nil
	}

	var err error
	p.addrIndex, p.sessionless, err = p.tryConnect(p.central.factory.SessionlessConnectTo, p.addrIndex, 0)
	return p.sessionless, err
}

func (p *PeerTransport) getSessionfulSmallTransport() (l1.OutTransport, error) {
	p.mutex.RLock()
	if t := p.sessionful; t != nil {
		p.mutex.RUnlock()
		return t, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.sessionful != nil {
		return p.sessionful, nil
	}
	if t, err := p._newSessionfulTransport(true); err != nil {
		return nil, err
	} else {
		p.sessionful = t
		return t, nil
	}
}

func (p *PeerTransport) getSessionfulLargeTransport() (l1.OutTransport, error) {
	p.mutex.RLock()
	if t := p._getSessionfulTransport(false); t != nil {
		p.mutex.RUnlock()
		return t, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p._newSessionfulTransport(false)
}

func (p *PeerTransport) _getSessionfulTransport(limitedLength bool) l1.OutTransport {
	for _, s := range p.connections {
		if o, ok := s.(l1.OutTransport); ok && TransportStreamFormat(o.GetTag()).IsDefinedLimited() == limitedLength {
			return o
		}
	}
	return nil
}

func (p *PeerTransport) _newSessionfulTransport(limitedLength bool) (t l1.OutTransport, err error) {
	if t = p._getSessionfulTransport(limitedLength); t != nil {
		return
	}

	limit := p.central.getTransportStreamFormat(limitedLength, false)

	p.addrIndex, t, err = p.tryConnect(p.central.factory.SessionfulConnectTo, p.addrIndex, limit)
	if err != nil {
		return nil, err
		//p.addConnection(t)
	}

	p.connections = append(p.connections, t)
	if semi, ok := t.(l1.SemiTransport); ok {
		go semi.ConnectReceiver(nil)
	}

	return
}

func (p *PeerTransport) addConnection(s io.Closer) {
	p.mutex.Lock()
	p.connections = append(p.connections, s)
	p.mutex.Unlock()
}

func (p *PeerTransport) addReceiver(conn io.Closer, incomingConnection bool) (TransportStreamFormat, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if incomingConnection {
		if p.connCount >= p.central.maxPeerConn {
			return 0, errors.New("connection limit exceeded")
		}

		p.connCount++
		p.connections = append(p.connections, conn)
		return DetectByFirstPacket, nil
	}

	for _, s := range p.connections {
		if oc, ok := s.(l1.TwoWayTransport); ok && oc.TwoWayConn() == conn {
			limit := TransportStreamFormat(oc.GetTag())
			if limit.IsDefined() {
				p.connCount++
				p.connections = append(p.connections, s)
				return limit, nil
			}
			break
		}
	}

	_, _, _ = p._resetConnection(conn, false)
	return 0, throw.Impossible()
}

func (p *PeerTransport) removeReceiver(conn io.Closer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.connCount--
	_, _, _ = p._resetConnection(conn, false)
}

type transportGetterFunc = func() (l1.OutTransport, error)

func (p *PeerTransport) useTransport(getTransportFn transportGetterFunc, canRetry bool, applyFn func(l1.OutTransport) error) error {
	for {
		if err := p.checkActive(); err != nil {
			return err
		}

		t, err := getTransportFn()
		if err != nil {
			return err
		}
		if err = applyFn(t); err == nil {
			return nil
		}
		if ok, hasMore := p.resetTransport(t, true); canRetry && ok && hasMore {
			continue
		}
		return err
	}
}

func (p *PeerTransport) EnsureConnect() error {
	// no need for p.smallMutex as there will be no write/send
	return p.useTransport(p.getSessionfulSmallTransport, true, func(l1.OutTransport) error {
		return nil
	})
}

//func (p *PeerTransport) UseSessionlessNoQuota(canRetry bool, applyFn OutFunc) error {
//	return p.useTransport(func() (l1.OutTransport, error) {
//		if t, err := p.getSessionlessTransport(); err != nil {
//			return nil, err
//		} else {
//			return t.WithQuota(nil), err
//		}
//	}, canRetry, applyFn)
//}

func (p *PeerTransport) UseSessionless(canRetry bool, applyFn uniproto.OutFunc) error {
	return p.useTransport(p.getSessionlessTransport, canRetry, applyFn)
}

func (p *PeerTransport) UseSessionful(size int64, canRetry bool, applyFn uniproto.OutFunc) error {
	if size <= uniproto.MaxNonExcessiveLength {
		p.smallMutex.Lock()
		defer p.smallMutex.Unlock()
		return p.useTransport(p.getSessionfulSmallTransport, canRetry, applyFn)
	}

	p.largeMutex.Lock()
	defer p.largeMutex.Unlock()
	return p.useTransport(p.getSessionfulLargeTransport, canRetry, applyFn)
}

func (p *PeerTransport) UseAny(size int64, canRetry bool, applyFn uniproto.OutFunc) error {
	if size <= int64(p.central.maxSessionlessSize) {
		return p.UseSessionless(canRetry, applyFn)
	}
	return p.UseSessionful(size, canRetry, applyFn)
}

func (p *PeerTransport) setAddresses(primary nwapi.Address, aliases []nwapi.Address) {
	switch {
	case primary.IsZero():
		panic(throw.IllegalValue())
	case len(aliases) >= math.MaxUint8:
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.aliases != nil {
		panic(throw.IllegalState())
	}

	p.aliases = append(make([]nwapi.Address, 0, 1+len(aliases)), primary)
	for i, aa := range aliases {
		if aa == primary {
			p.aliases = append(p.aliases, aliases[:i]...)
			p.aliases = append(p.aliases, aliases[i+1:]...)
			return
		}
	}
	p.aliases = append(p.aliases, aliases...)
}

func (p *PeerTransport) addAliases(aliases []nwapi.Address) {
	n := len(aliases)
	if n == 0 {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if n+len(p.aliases) > math.MaxUint8 {
		panic(throw.IllegalValue())
	}
	p.aliases = append(p.aliases, aliases...)
}

func (p *PeerTransport) removeAlias(a nwapi.Address) bool {
	if a.IsZero() {
		return false
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i, aa := range p.aliases {
		if i != 0 && aa == a {
			copy(p.aliases[i:], p.aliases[i+1:])
			p.aliases = p.aliases[:len(p.aliases)-1]
			return true
		}
	}
	return false
}
