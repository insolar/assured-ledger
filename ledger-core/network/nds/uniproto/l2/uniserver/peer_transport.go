// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"errors"
	"io"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi/nwaddr"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/ratelimiter"
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
	factory    PeerTransportFactory
	sigFactory PeerCryptographyFactory

	maxSessionlessSize uint16
	maxPeerConn        uint8
	preferHTTP         bool
	retryLimit         uint8
	retryDelayInc      time.Duration
	retryDelayVariance time.Duration
	retryDelayMax      time.Duration
}

func (p *PeerTransportCentral) checkActive(deadPeer bool) error {
	if deadPeer || !p.factory.IsActive() {
		return errors.New("aborted")
	}
	return nil
}

func (p *PeerTransportCentral) getTransportStreamFormat(limitedLength bool, peerPreferHTTP bool) TransportStreamFormat {
	switch {
	case !p.preferHTTP && !peerPreferHTTP:
		if limitedLength {
			return BinaryLimitedLength
		}
		return BinaryUnlimitedLength
	case limitedLength:
		return HTTPLimitedLength
	default:
		return HTTPUnlimitedLength
	}
}

func (p *PeerTransportCentral) incRetryDelay(delay time.Duration) time.Duration {
	if delay >= p.retryDelayMax {
		return delay
	}
	return delay + p.retryDelayInc + time.Duration(rand.Int63n(int64(p.retryDelayVariance)))
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

// nolint:interfacer
func (p *PeerTransport) discardTransportAndAddress(t l1.OutTransport, changeAddress bool) (hasMore bool) {
	if t == nil {
		return false
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p._resetConnection(t, changeAddress)

	if !changeAddress {
		return true
	}

	if p.addrIndex++; int(p.addrIndex) >= len(p.aliases) {
		p.addrIndex = 0
		return false
	}
	return true
}

func (p *PeerTransport) _resetConnection(t io.Closer, changeAddress bool) {
	if t == p.sessionless {
		p.sessionless = nil
		_ = t.Close()
		return
	}

	if t == p.sessionful {
		// Sessionless has to follow a sessionful connection
		if s := p.sessionless; changeAddress && s != nil {
			p.sessionless = nil
			_ = s.Close()
		}
		p.sessionful = nil
	}

	j := 0
	for i := range p.connections {
		s := p.connections[i]
		if s == t {
			continue
		}
		if nout, ok := s.(l1.TwoWayTransport); ok && nout.TwoWayConn() == t {
			_ = s.Close()
			continue
		}
		if i != j {
			p.connections[j] = s
		}
		j++
	}

	switch {
	case j == 0:
		p.connections = nil
	default:
		p.connections = p.connections[:j]
	}
	_ = t.Close()
}

type connectFunc = func(to nwapi.Address) (l1.OutTransport, error)

func (p *PeerTransport) tryConnect(factoryFn connectFunc, limit TransportStreamFormat) (l1.OutTransport, error) {

	startIndex := p.addrIndex
	for {
		addr := p.aliases[p.addrIndex]

		if p.addrIndex == 0 && addr.IdentityType() == nwaddr.DNS {
			// primary address is always resolved and can be ignored when is DNS
			// then there MUST be more addresses
			p.addrIndex++
			if startIndex == 1 {
				return nil, nil
			}
			continue
		}

		if err := p.checkActive(); err != nil {
			return nil, err
		}

		switch t, err := factoryFn(addr); {
		case err != nil:
			//
		case t == nil:
			// non-connectable
			if p.addrIndex++; int(p.addrIndex) >= len(p.aliases) {
				p.addrIndex = 0
			}
			if p.addrIndex == startIndex {
				return nil, nil
			}
		case p.rateQuota != nil:
			t = t.WithQuota(p.rateQuota.WriteBucket())
			fallthrough
		default:
			t.SetTag(int(limit))
			return t, nil
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
	p.sessionless, err = p.tryConnect(p.central.factory.SessionlessConnectTo, 0)
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
	t, err := p._newSessionfulTransport(true)
	if err != nil {
		return nil, err
	}
	p.sessionful = t
	return t, nil
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

	t, err = p.tryConnect(p.central.factory.SessionfulConnectTo, limit)
	if err != nil || t == nil {
		return
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

	p._resetConnection(conn, false)
	return 0, throw.Impossible()
}

func (p *PeerTransport) removeReceiver(conn io.Closer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.connCount--
	p._resetConnection(conn, false)
}

type transportGetterFunc = func() (l1.OutTransport, error)

func (p *PeerTransport) useTransport(getTransportFn transportGetterFunc, sessionfulTransport bool, applyFn uniproto.OutFunc) error {
	var delay time.Duration

	repeatedNoAddress := false
	for i := int(p.central.retryLimit) + 1; i >= 0; i-- {
		if err := p.checkActive(); err != nil {
			return err
		}

		t, err := getTransportFn()
		switch {
		case err != nil:
			return err
		case t == nil:
			// no more addresses
			if repeatedNoAddress {
				panic(throw.IllegalState())
			}
			repeatedNoAddress = true
			delay = p.central.incRetryDelay(delay)
			time.Sleep(delay)
			continue
		default:
			repeatedNoAddress = false
			canRetry := false

			facade := outFacade{delegate: t}
			switch canRetry, err = applyFn(&facade); {
			case err == nil:
				return nil
			case !facade.sentData:
				// transport wasn't in use
				return err
			case !facade.hadError:
				if sessionfulTransport {
					// this is not a transport error, but something might be sent
					// so have to reset a sessionful connection
					p.discardTransportAndAddress(t, false)
				}
				return err
			case !canRetry:
				p.discardTransportAndAddress(t, true)
				return err
			}

			lastDelay := delay
			delay = p.central.incRetryDelay(delay)
			if !p.discardTransportAndAddress(t, true) {
				lastDelay = delay
			}
			time.Sleep(lastDelay)

		}
	}
	return throw.FailHere("retry limit exceeded")
}

func (p *PeerTransport) EnsureConnect() error {
	// no need for p.smallMutex as there will be no write/send
	return p.useTransport(p.getSessionfulSmallTransport, true,
		func(l1.BasicOutTransport) (bool, error) { return true, nil })
}

func (p *PeerTransport) UseSessionless(applyFn uniproto.OutFunc) error {
	return p.useTransport(p.getSessionlessTransport, false, applyFn)
}

func (p *PeerTransport) UseSessionful(size int64, applyFn uniproto.OutFunc) error {
	if size <= uniproto.MaxNonExcessiveLength {
		p.smallMutex.Lock()
		defer p.smallMutex.Unlock()
		return p.useTransport(p.getSessionfulSmallTransport, true, applyFn)
	}

	p.largeMutex.Lock()
	defer p.largeMutex.Unlock()
	return p.useTransport(p.getSessionfulLargeTransport, true, applyFn)
}

func (p *PeerTransport) CanUseSessionless(size int64) bool {
	return size <= int64(p.central.maxSessionlessSize)
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

	p.aliases = append(make([]nwapi.Address, 0, len(aliases)+2), primary, nwapi.NewLocalUID(p.uid, 0))
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

func (p *PeerTransport) sendPacket(tp uniproto.OutType, fn uniproto.OutFunc) error {
	switch tp {
	case uniproto.SessionfulLarge:
		p.largeMutex.Lock()
		defer p.largeMutex.Unlock()
		return p.useTransport(p.getSessionfulLargeTransport, true, fn)

	case uniproto.SessionfulSmall:
		p.smallMutex.Lock()
		defer p.smallMutex.Unlock()
		return p.useTransport(p.getSessionfulSmallTransport, true, fn)

	case uniproto.Sessionless:
		return p.useTransport(p.getSessionlessTransport, false, fn)

	case uniproto.SessionlessNoQuota:
		if p.rateQuota == nil {
			return p.useTransport(p.getSessionlessTransport, false, fn)
		}

		return p.useTransport(func() (t l1.OutTransport, err error) {
			if t, err = p.getSessionlessTransport(); t == nil {
				return
			}
			return t.WithQuota(nil), err
		}, false, fn)

	default:
		panic(throw.Impossible())
	}
}

func (p *PeerTransport) limitReader(c io.ReadCloser) io.ReadCloser {
	if p.rateQuota != nil {
		return iokit.RateLimitReader(c, p.rateQuota)
	}
	return c
}

func (p *PeerTransport) limitWriter(c io.WriteCloser) io.WriteCloser {
	if p.rateQuota != nil {
		return iokit.RateLimitWriter(c, p.rateQuota.WriteBucket())
	}
	return c
}
