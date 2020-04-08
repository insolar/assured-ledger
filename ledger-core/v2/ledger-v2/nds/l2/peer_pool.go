// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"crypto/tls"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewPeerManager(factory PeerTransportFactory, local l1.Address, localFn func(*Peer)) *PeerManager {
	if factory == nil {
		panic(throw.IllegalValue())
	}
	pm := &PeerManager{}
	pm.central.factory = factory

	if err := pm.addLocal(local, nil, func(peer *Peer) error {
		if localFn != nil {
			localFn(peer)
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return pm
}

type PeerManager struct {
	quotaFactory PeerQuotaFactoryFunc
	svFactory    cryptkit.DataSignatureVerifierFactory
	peerFactory  OfflinePeerFactoryFunc

	central PeerTransportCentral
	// LOCK: WARNING! PeerTransport.mutex can be acquired under this mutex
	peerMutex  sync.RWMutex
	peers      []*Peer
	peerUnused int
	aliases    map[l1.Address]uint32
}

type PeerQuotaFactoryFunc = func([]l1.Address) ratelimiter.RWRateQuota
type OfflinePeerFactoryFunc = func(*Peer) error

func (p *PeerManager) _ensureEmpty() {
	if len(p.aliases) > 0 {
		panic(throw.IllegalState())
	}
}

func (p *PeerManager) SetQuotaFactory(quotaFn PeerQuotaFactoryFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p._ensureEmpty()
	p.quotaFactory = quotaFn
}

func (p *PeerManager) SetPeerFactory(fn OfflinePeerFactoryFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p._ensureEmpty()
	p.peerFactory = fn
}

func (p *PeerManager) SetVerifierFactory(f cryptkit.DataSignatureVerifierFactory) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p._ensureEmpty()
	p.svFactory = f
}

func (p *PeerManager) peer(a l1.Address) (uint32, *Peer) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peer(a)
}

func (p *PeerManager) peerNotLocal(a l1.Address) (*Peer, error) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peerNotLocal(a)
}

func (p *PeerManager) _peerNotLocal(a l1.Address) (*Peer, error) {
	if idx, peer := p._peer(a); idx == 0 && peer != nil {
		return nil, throw.Violation("loopback is not allowed")
	} else {
		return peer, nil
	}
}

func (p *PeerManager) _peer(a l1.Address) (uint32, *Peer) {
	if idx, ok := p.aliases[a.HostOnly()]; ok {
		return idx, p.peers[idx]
	}
	return 0, nil
}

func (p *PeerManager) _aliasesFor(peer *Peer, peerIndex uint32, aliases []l1.Address) ([]l1.Address, error) {
	j := 0
	for i, a := range aliases {
		switch conflictIndex, hasConflict := p.aliases[a.HostOnly()]; {
		case !hasConflict:
			if i != j {
				aliases[j] = a
			}
			j++
		case conflictIndex == peerIndex && peer != nil:
			//
		default:
			var primary l1.Address
			if peer != nil {
				primary = peer.GetPrimary()
			}
			return nil, throw.E("alias conflict", struct {
				Address, Alias, ConflictWith l1.Address
			}{primary, a, p.peers[conflictIndex].GetPrimary()})
		}
	}
	return aliases[:j], nil
}

func (p *PeerManager) HasAddress(a l1.Address) bool {
	return !p.GetPrimary(a).IsZero()
}

func (p *PeerManager) Local() *Peer {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()
	return p.peers[0]
}

func (p *PeerManager) GetPrimary(a l1.Address) l1.Address {
	if _, peer := p.peer(a); peer != nil {
		return peer.GetPrimary()
	}
	return l1.Address{}
}

func (p *PeerManager) RemovePeer(a l1.Address) bool {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peerIdx, peer := p._peer(a)
	switch {
	case peer == nil:
		return false
	case peerIdx == 0:
		panic(throw.Impossible())
	}

	for _, aa := range peer.onRemoved() {
		delete(p.aliases, aa.HostOnly())
	}
	p.peers[peerIdx] = nil
	p.peerUnused++
	return true
}

func (p *PeerManager) RemoveAlias(a l1.Address) bool {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	_, peer := p._peer(a)
	if peer == nil {
		return false
	}

	if peer.removeAlias(a) {
		delete(p.aliases, a)
		return true
	}
	return false
}

func (p *PeerManager) addLocal(primary l1.Address, aliases []l1.Address, newPeerFn func(*Peer) error) error {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p._ensureEmpty()
	if primary.IsZero() {
		panic(throw.IllegalValue())
	}

	_, _, err := p._newPeer(newPeerFn, primary, aliases)
	return err
}

func (p *PeerManager) AddPeer(primary l1.Address, aliases ...l1.Address) {
	if err := p.addPeer(primary, aliases...); err != nil {
		panic(err)
	}
}

func (p *PeerManager) addPeer(primary l1.Address, aliases ...l1.Address) error {
	if primary.IsZero() {
		panic(throw.IllegalValue())
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	if _, _, err := p._newPeer(nil, primary, aliases); err != nil {
		return err
	}
	return nil
}

func (p *PeerManager) AddAliases(to l1.Address, aliases ...l1.Address) {
	if err := p.addAliases(to, aliases...); err != nil {
		panic(err)
	}
}

func (p *PeerManager) addAliases(to l1.Address, aliases ...l1.Address) error {
	peerIndex, peer := p.peer(to)
	if peer == nil {
		return throw.E("unknown peer", struct{ l1.Address }{to})
	}

	if len(aliases) == 0 {
		return nil
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	switch aliases, err := p._aliasesFor(peer, peerIndex, aliases); {
	case err != nil:
		return err
	case len(aliases) == 0:
		//
	default:
		peer.transport.AddAliases(aliases)
		for _, a := range aliases {
			p.aliases[a.HostOnly()] = peerIndex
		}
	}
	return nil
}

func (p *PeerManager) _newPeer(newPeerFn func(*Peer) error, primary l1.Address, aliases []l1.Address) (uint32, *Peer, error) {
	peer := &Peer{}
	peer.transport.central = &p.central
	peer.transport.SetAddresses(primary, aliases)

	if p.peerFactory != nil {
		if err := p.peerFactory(peer); err != nil {
			return 0, nil, err
		}
	}

	if err := newPeerFn(peer); err != nil {
		return 0, nil, err
	}

	if _, err := p._aliasesFor(nil, 0, peer.transport.aliases); err != nil {
		return 0, nil, err
	}

	if p.quotaFactory != nil {
		peer.transport.rateQuota = p.quotaFactory(aliases)
	}

	peerIndex := uint32(len(p.peers))
	// TODO reuse unallocated
	p.peers = append(p.peers, peer)
	for _, a := range aliases {
		p.aliases[a.HostOnly()] = peerIndex
	}

	return peerIndex, peer, nil
}

func (p *PeerManager) Close() error {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()
	p.aliases = nil
	peers := p.peers
	p.peers = nil
	for _, p := range peers {
		if p != nil {
			p.onRemoved()
		}
	}
	return nil
}

func (p *PeerManager) connectFrom(remote l1.Address, newPeerFn func(*Peer) error) (*Peer, error) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peer, err := p._peerNotLocal(remote)
	if err == nil && peer == nil {
		_, peer, err = p._newPeer(newPeerFn, remote, nil)
	}
	return peer, err
}

func (p *PeerManager) AddHostId(to l1.Address, id apinetwork.HostId) bool {
	if id == 0 {
		return false
	}
	return p.addAliases(to, l1.NewHostId(id)) == nil
}

/**************************************/

type PeerState uint8

const (
	_ PeerState = iota
	//Established
	Connected
	Verified
	Trusted
)

type Peer struct {
	transport PeerTransport
	mutex     sync.Mutex
	pk        cryptkit.SignatureKey
	dsv       cryptkit.DataSignatureVerifier

	id    apinetwork.HostId
	state atomickit.Uint32 // PeerState

	// HostIds for indirectly accessible hosts
}

func (p *Peer) GetPrimary() l1.Address {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	if aliases := p.transport.aliases; len(aliases) > 0 {
		return aliases[0]
	}
	return l1.Address{}
}

func (p *Peer) onRemoved() []l1.Address {
	return p.transport.kill()
}

func (p *Peer) verifyByTls(_ *tls.Conn) (verified bool, err error) {
	return false, nil
}

func (p *Peer) SetSignatureKey(pk cryptkit.SignatureKey) {
	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	p.pk = pk
	p.dsv = nil
}

func (p *Peer) GetSignatureVerifier(factory cryptkit.DataSignatureVerifierFactory) (cryptkit.DataSignatureVerifier, error) {
	p.transport.mutex.RLock()
	if dsv := p.dsv; dsv != nil {
		p.transport.mutex.RUnlock()
		return dsv, nil
	}
	p.transport.mutex.RUnlock()

	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	switch {
	case p.dsv != nil:
		return p.dsv, nil
	case p.pk.IsEmpty():
		return nil, throw.E("peer key is unknown")
	//case !factory.IsSignatureKeySupported(p.pk):
	default:
		dsv := factory.CreateDataSignatureVerifier(p.pk)
		if dsv == nil {
			return nil, throw.E("unsupported peer key")
		}
		p.dsv = dsv
		return dsv, nil
	}
}

func (p *Peer) UpgradeState(state PeerState) {
	p.state.SetGreater(uint32(state))
}

func (p *Peer) getState() PeerState {
	return PeerState(p.state.Load())
}

func (p *Peer) HasState(state PeerState) bool {
	return p.getState() >= state
}

func (p *Peer) checkVerified() error {
	if p.HasState(Verified) {
		return nil
	}
	return throw.Violation("peer is not trusted")
}

func (p *Peer) removeAlias(a l1.Address) bool {
	return p.transport.RemoveAliases(a)
}
