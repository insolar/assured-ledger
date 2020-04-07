// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"crypto/tls"
	"net"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewPeerManager(factory PeerTransportFactory) *PeerManager {
	if factory == nil {
		panic(throw.IllegalValue())
	}
	pm := &PeerManager{}
	pm.central.factory = factory
	return pm
}

type PeerManager struct {
	quotaFactory PeerQuotaFactoryFunc
	sigvFactory  cryptkit.SignatureVerifierFactory

	central    PeerTransportCentral
	peerMutex  sync.RWMutex
	peers      []*Peer
	peerUnused int
	aliases    map[l1.Address]uint32
}
type PeerQuotaFactoryFunc func([]l1.Address) ratelimiter.RWRateQuota

func (p *PeerManager) SetQuotaFactory(quotaFn PeerQuotaFactoryFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	if len(p.aliases) > 0 {
		panic(throw.IllegalState())
	}
	p.quotaFactory = quotaFn
}

func (p *PeerManager) peer(a l1.Address) (uint32, *Peer) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peer(a)
}

func (p *PeerManager) peerNoLocal(a l1.Address) (*Peer, error) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

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
	if len(p.aliases) == 0 {
		panic(throw.IllegalState())
	}
	return p.peers[0]
}

func (p *PeerManager) GetPrimary(a l1.Address) l1.Address {
	if _, peer := p.peer(a); peer != nil {
		return peer.GetPrimary()
	}
	return l1.Address{}
}

func (p *PeerManager) Remove(a l1.Address) (bool, error) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peerIdx, peer := p._peer(a)
	switch {
	case peer == nil:
		return false, nil
	case peerIdx == 0:
		return false, throw.Impossible()
	}

	for _, aa := range peer.transport.aliases {
		delete(p.aliases, aa.HostOnly())
	}
	p.peers[peerIdx] = nil
	p.peerUnused++
	return true, peer.onRemoved()
}

func (p *PeerManager) AddLocal(primary l1.Address, aliases []l1.Address, newPeerFn func(*Peer) error) error {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	switch {
	case primary.IsZero():
		panic(throw.IllegalValue())
	case len(p.aliases) != 0:
		panic(throw.IllegalState())
	}

	_, _, err := p._newPeer(newPeerFn, primary, aliases)
	return err
}

func (p *PeerManager) Add(primary l1.Address, aliases ...l1.Address) error {
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

func (p *PeerManager) AddAliases(to l1.Address, aliases ...l1.Address) error {
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

	if err := newPeerFn(peer); err != nil {
		return 0, nil, err
	}

	var err error
	if _, err = p._aliasesFor(nil, 0, peer.transport.aliases); err != nil {
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
			_ = p.onRemoved()
		}
	}
	return nil
}

func (p *PeerManager) connectFrom(remote l1.Address, newPeerFn func(*Peer) error) (*Peer, error) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peer, err := p.peerNoLocal(remote)
	if err == nil && peer == nil {
		_, peer, err = p._newPeer(newPeerFn, remote, nil)
	}
	return peer, err
}

/**************************************/

type Peer struct {
	transport PeerTransport
	pk        cryptkit.SignatureKey

	// peerState
	// PK

	// TODO hashing/signature verification support
	// TODO indirection support
}

func (p *Peer) GetPrimary() l1.Address {
	if aliases := p.transport.aliases; len(aliases) > 0 {
		return aliases[0]
	}
	return l1.Address{}
}

func (p *Peer) onRemoved() error {
	return p.transport.kill()
}

func (p *Peer) VerifyByTls(*tls.Conn) bool {
	return false
}

func (p *Peer) getVerifier(apinetwork.ProtocolSupporter, *apinetwork.Header, cryptkit.SignatureVerifierFactory) (cryptkit.DataSignatureVerifier, error) {
	return p.pk
}

/**************************************/

var _ UnifiedPeer = unifiedPeer{}

type unifiedPeer struct {
	mgr   *PeerManager
	peer  *Peer
	index uint32
}

func (p unifiedPeer) VerifyAddress(remote net.Addr) (l1.Address, apinetwork.VerifyHeaderFunc, error) {
	panic("implement me")
}

func (unifiedPeer) UnifiedPeer() {}
