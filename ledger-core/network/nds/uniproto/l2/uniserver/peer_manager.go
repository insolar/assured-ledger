// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/ratelimiter"
)

func NewPeerManager(factory PeerTransportFactory, local nwapi.Address, localFn func(*Peer)) *PeerManager {
	if factory == nil {
		panic(throw.IllegalValue())
	}
	pm := &PeerManager{}
	pm.central.factory = factory
	pm.central.maxPeerConn = 4

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

// PeerManager runs a map of peers. See PeerMap
type PeerManager struct {
	quotaFactoryFn PeerQuotaFactoryFunc
	peerMapperFn   PeerMapperFunc

	central PeerTransportCentral
	// LOCK: WARNING! PeerTransport.connMutex can be acquired under this mutex
	peerMutex sync.RWMutex
	peers     PeerMap
}

// PeerQuotaFactoryFunc is invoked to create a transfer rate quota for a peer. Quota applies to all connections of the peer. Can return nil.
type PeerQuotaFactoryFunc = func([]nwapi.Address) ratelimiter.RWRateQuota
// PeerMapperFunc is invoked after a new peer is registered to check if this peek has to be merged with another one.
type PeerMapperFunc = func(*Peer) (remapTo nwapi.Address, err error)

// SetIdentityClassifier can only be used for initialization
func (p *PeerManager) SetIdentityClassifier(idWithPortFn func(nwapi.Address) bool) {
	p.peers.idWithPortFn = idWithPortFn
}

// SetPeerConnectionLimit sets a max number of sessionful connections established to/from peer.
func (p *PeerManager) SetPeerConnectionLimit(n uint8) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.central.maxPeerConn = n
}

// SetQuotaFactory sets factory to allocate traffic quota when a peer is added.
func (p *PeerManager) SetQuotaFactory(quotaFn PeerQuotaFactoryFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	p.quotaFactoryFn = quotaFn
}

func (p *PeerManager) SetPeerMapper(fn PeerMapperFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	p.peerMapperFn = fn
}

// SetSignatureFactory sets a provider of cryptographic tools. Changing this factory will not unset already created cryptographic tools.
func (p *PeerManager) SetSignatureFactory(f PeerCryptographyFactory) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	p.central.sigFactory = f
}

func (p *PeerManager) peer(a nwapi.Address) (uint32, *Peer) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p.peers.get(a)
}

func (p *PeerManager) peerNilLocal(a nwapi.Address) (*Peer, error) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peerNilLocal(a)
}

func (p *PeerManager) _peerNilLocal(a nwapi.Address) (*Peer, error) {
	switch idx, peer := p.peers.get(a); {
	case peer == nil:
		return nil, throw.FailHere("unknown peer")
	case idx == 0:
		return nil, nil
	default:
		return peer, nil
	}
}

func (p *PeerManager) peerNotLocal(a nwapi.Address) (*Peer, error) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peerNotLocal(a)
}

func (p *PeerManager) _peerNotLocal(a nwapi.Address) (*Peer, error) {
	idx, peer := p.peers.get(a)
	if idx == 0 && peer != nil {
		return nil, throw.Violation("loopback is not allowed")
	}
	return peer, nil
}

// HasAddress return true when there is a peer registered with the given address or alias.
func (p *PeerManager) HasAddress(a nwapi.Address) bool {
	return !p.GetPrimary(a).IsZero()
}

// Local returns a local peer - peer that represents a local node.
func (p *PeerManager) Local() *Peer {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()
	return p.peers.peers[0]
}

// GetPrimary returns primary address of a peer registered with the given address or alias. Will return zero value when not found.
func (p *PeerManager) GetPrimary(a nwapi.Address) nwapi.Address {
	if _, peer := p.peer(a); peer != nil {
		return peer.GetPrimary()
	}
	return nwapi.Address{}
}

// RemovePeer removes/unregisters a peer with the given address or alias.
func (p *PeerManager) RemovePeer(a nwapi.Address) bool {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peerIdx, peer := p.peers.get(a)
	switch {
	case peer == nil:
		return false
	case peerIdx == 0:
		panic(throw.Impossible())
	}

	p.peers.remove(peerIdx)
	return true
}

// RemoveAlias removes/unregisters the given alias. Will return false when the alias is unknown or can't be unregistered (is primary).
func (p *PeerManager) RemoveAlias(a nwapi.Address) bool {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	return p.peers.removeAlias(a)
}

func (p *PeerManager) addLocal(primary nwapi.Address, aliases []nwapi.Address, newPeerFn func(*Peer) error) error {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	if primary.IsZero() {
		panic(throw.IllegalValue())
	}

	_, _, err := p._newPeer(newPeerFn, primary, aliases)
	return err
}

func (p *PeerManager) updateLocalPrimary(primary nwapi.Address) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()
	p.peers.updatePrimary(primary, 0)
}

// MustAddPeer creates a peer for the given primary address and aliases. Panics on collisions.
func (p *PeerManager) MustAddPeer(primary nwapi.Address, aliases ...nwapi.Address) {
	if err := p.addPeer(primary, aliases...); err != nil {
		panic(err)
	}
}

func (p *PeerManager) addPeer(primary nwapi.Address, aliases ...nwapi.Address) error {
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

// MustAddAliases adds aliases for a peer identified by (to). Panics on an unknown peer or on collisions.
func (p *PeerManager) MustAddAliases(to nwapi.Address, aliases ...nwapi.Address) {
	if err := p.addAliases(to, aliases...); err != nil {
		panic(err)
	}
}

func (p *PeerManager) getPeer(a nwapi.Address) (uint32, *Peer, error) {
	peerIndex, peer := p.peer(a)
	if peer == nil {
		return 0, nil, throw.E("unknown peer", struct{ nwapi.Address }{a})
	}
	return peerIndex, peer, nil
}

func (p *PeerManager) addAliases(to nwapi.Address, aliases ...nwapi.Address) error {
	peerIndex, _, err := p.getPeer(to)
	if err != nil {
		return err
	}

	if len(aliases) == 0 {
		return nil
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	return p.peers.addAliases(peerIndex, aliases)
}

func (p *PeerManager) _newPeer(newPeerFn func(*Peer) error, primary nwapi.Address, aliases []nwapi.Address) (peer *Peer, peerIdx uint32, err error) {
	peer = &Peer{}
	peer.transport.uid = NextPeerUID()
	peer.transport.central = &p.central
	peer.transport.setAddresses(primary, aliases)

	remapped := false
	if err = func() error {
		if p.peerMapperFn != nil && !p.peers.isEmpty() {
			switch remapTo, err := p.peerMapperFn(peer); {
			case err != nil:
				return err
			case !remapTo.IsZero():
				switch idx, remapPeer := p.peers.get(remapTo); {
				case remapPeer == nil:
					peer.transport.addAliases([]nwapi.Address{remapTo})
				case idx == 0:
					return throw.FailHere("remap to loopback")
				default:
					moreAliases := make([]nwapi.Address, 0, 1+len(aliases))
					moreAliases = append(moreAliases, primary)
					moreAliases = append(moreAliases, aliases...)
					if err := p.peers.addAliases(idx, moreAliases); err != nil {
						return err
					}
					peer = remapPeer
					remapped = true
					return nil
				}
			}
		}

		if newPeerFn != nil {
			if err := newPeerFn(peer); err != nil {
				return err
			}
		}

		if !peer.nodeID.IsAbsent() {
			peer.transport.addAliases([]nwapi.Address{nwapi.NewHostID(nwapi.HostID(peer.nodeID))})
		}

		peer.transport.aliases, err = p.peers.checkAliases(nil, math.MaxUint32, peer.transport.aliases)
		return err

	}(); err != nil {
		return nil, 0, err
	}
	if remapped {
		return
	}

	if p.quotaFactoryFn != nil {
		peer.transport.rateQuota = p.quotaFactoryFn(peer.transport.aliases)
	}

	peerIdx = p.peers.addPeer(peer)
	return peer, peerIdx, nil
}

// Close removes all peers with relevant finalization of them.
func (p *PeerManager) Close() error {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()
	for _, p := range p.peers.cleanup() {
		if p != nil {
			p.onRemoved()
		}
	}
	return nil
}

func (p *PeerManager) connectionFrom(remote nwapi.Address, newPeerFn func(*Peer) error) (*Peer, error) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peer, err := p._peerNotLocal(remote)
	if err == nil && peer == nil {
		peer, _, err = p._newPeer(newPeerFn, remote, nil)
	}
	return peer, err
}

// AddHostID adds an alias with protocol specific HostID. When HostID is NodeID then will also call SetNodeID
func (p *PeerManager) AddHostID(to nwapi.Address, id nwapi.HostID) (bool, error) {
	peerIndex, peer, err := p.getPeer(to)
	if err != nil {
		return false, err
	}
	if id == 0 {
		return false, nil
	}

	if id.IsNodeID() {
		peer.SetNodeID(id.AsNodeID())
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	err = p.peers.addAliases(peerIndex, []nwapi.Address{nwapi.NewHostID(id)})
	return err == nil, err
}

func (p *PeerManager) connectPeer(remote nwapi.Address) (*Peer, error) {
	switch peer, err := p.peerNilLocal(remote); {
	case err != nil:
		// not connected
	case peer != nil:
		return peer, nil
	default:
		// local
		return nil, nil
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peer, err := p._peerNilLocal(remote)
	switch {
	case peer != nil:
		return peer, nil
	case err == nil:
		// local
		return nil, nil

	}

	var peerIdx uint32
	if peer, peerIdx, err = p._newPeer(nil, remote, nil); err != nil {
		return nil, err
	}

	if err = peer.transport.EnsureConnect(); err != nil {
		p.peers.remove(peerIdx)
		return nil, err
	}
	return peer, nil
}

// GetLocalDataDecrypter returns Decrypter for the local peer.
func (p *PeerManager) GetLocalDataDecrypter() (cryptkit.Decrypter, error) {
	return p.Local().getDataDecrypter()
}

func (p *PeerManager) Manager() uniproto.PeerManager {
	return facadePeerManager{
		peerManager:    p,
		maxSmall:       uniproto.MaxNonExcessiveLength,
	}
}

/****************************/

var _ uniproto.PeerManager = &facadePeerManager{}

type facadePeerManager struct {
	peerManager    *PeerManager
	maxSmall       uint16
}

func (v facadePeerManager) ConnectPeer(address nwapi.Address) (uniproto.Peer, error) {
	switch peer, err := v.peerManager.connectPeer(address); {
	case err != nil:
		return nil, err
	case peer == nil:
		return nil, nil
	default:
		return peer, nil
	}
}

func (v facadePeerManager) ConnectedPeer(address nwapi.Address) (uniproto.Peer, error) {
	switch peer, err := v.peerManager.peerNilLocal(address); {
	case err != nil:
		return nil, err
	case peer == nil:
		return nil, nil
	default:
		return peer, nil
	}
}

func (v facadePeerManager) LocalPeer() uniproto.Peer {
	return v.peerManager.Local()
}

func (v facadePeerManager) MaxSessionlessPayloadSize() uint {
	return uint(v.peerManager.central.factory.MaxSessionlessSize())
}

func (v facadePeerManager) MaxSmallPayloadSize() uint {
	return uint(v.maxSmall)
}
