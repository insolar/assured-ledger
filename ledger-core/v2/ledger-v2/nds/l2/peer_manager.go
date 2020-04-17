// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewPeerManager(factory PeerTransportFactory, local nwapi.Address, localFn func(*Peer)) *PeerManager {
	if factory == nil {
		panic(throw.IllegalValue())
	}
	pm := &PeerManager{}
	pm.central.factory = factory
	pm.central.maxSessionlessSize = factory.MaxSessionlessSize()
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

type PeerCryptographyFactory interface {
	cryptkit.DataSignatureVerifierFactory
	cryptkit.DataSignerFactory
	IsSignatureKeySupported(cryptkit.SignatureKey) bool
	CreateDataDecrypter(cryptkit.SignatureKey) cryptkit.Decrypter
	CreateDataEncrypter(cryptkit.SignatureKey) cryptkit.Encrypter
}

type PeerManager struct {
	quotaFactory PeerQuotaFactoryFunc
	peerFactory  OfflinePeerFactoryFunc

	central PeerTransportCentral
	// LOCK: WARNING! PeerTransport.mutex can be acquired under this mutex
	peerMutex sync.RWMutex
	peers     PeerMap
}

type PeerQuotaFactoryFunc = func([]nwapi.Address) ratelimiter.RWRateQuota
type OfflinePeerFactoryFunc = func(*Peer) error

func (p *PeerManager) SetPeerConnectionLimit(n uint8) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.central.maxPeerConn = n
}

func (p *PeerManager) SetMaxSessionlessSize(n uint16) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.central.maxSessionlessSize = n
}

func (p *PeerManager) SetQuotaFactory(quotaFn PeerQuotaFactoryFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	p.quotaFactory = quotaFn
}

func (p *PeerManager) SetPeerFactory(fn OfflinePeerFactoryFunc) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	p.peerFactory = fn
}

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

func (p *PeerManager) peerNotLocal(a nwapi.Address) (*Peer, error) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peerNotLocal(a)
}

func (p *PeerManager) _peerNotLocal(a nwapi.Address) (*Peer, error) {
	if idx, peer := p.peers.get(a); idx == 0 && peer != nil {
		return nil, throw.Violation("loopback is not allowed")
	} else {
		return peer, nil
	}
}

func (p *PeerManager) HasAddress(a nwapi.Address) bool {
	return !p.GetPrimary(a).IsZero()
}

func (p *PeerManager) Local() *Peer {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()
	return p.peers.peers[0]
}

func (p *PeerManager) GetPrimary(a nwapi.Address) nwapi.Address {
	if _, peer := p.peer(a); peer != nil {
		return peer.GetPrimary()
	}
	return nwapi.Address{}
}

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

func (p *PeerManager) AddPeer(primary nwapi.Address, aliases ...nwapi.Address) {
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

func (p *PeerManager) AddAliases(to nwapi.Address, aliases ...nwapi.Address) {
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

func (p *PeerManager) _newPeer(newPeerFn func(*Peer) error, primary nwapi.Address, aliases []nwapi.Address) (uint32, *Peer, error) {
	peer := &Peer{}
	peer.transport.uid = NextPeerUID()
	peer.transport.central = &p.central
	peer.transport.setAddresses(primary, aliases)
	aliases = peer.transport.aliases

	if p.peerFactory != nil {
		if err := p.peerFactory(peer); err != nil {
			return 0, nil, err
		}
	}

	if newPeerFn != nil {
		if err := newPeerFn(peer); err != nil {
			return 0, nil, err
		}
	}

	if _, err := p.peers.checkAliases(nil, math.MaxUint32, aliases); err != nil {
		return 0, nil, err
	}

	if p.quotaFactory != nil {
		peer.transport.rateQuota = p.quotaFactory(aliases)
	}

	return p.peers.addPeer(peer), peer, nil
}

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
		_, peer, err = p._newPeer(newPeerFn, remote, nil)
	}
	return peer, err
}

func (p *PeerManager) AddHostId(to nwapi.Address, id nwapi.HostID) (bool, error) {
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

	err = p.peers.addAliases(peerIndex, []nwapi.Address{nwapi.NewHostId(id)})
	return err == nil, err
}

func (p *PeerManager) connectPeer(remote nwapi.Address) (*Peer, error) {
	switch peer, err := p.peerNotLocal(remote); {
	case err != nil:
		return nil, err
	case peer != nil:
		return peer, nil
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peer, err := p._peerNotLocal(remote)
	if err == nil && peer == nil {
		_, peer, err = p._newPeer(nil, remote, nil)
	}
	if err != nil {
		return nil, err
	}
	if err = peer.transport.EnsureConnect(); err != nil {
		return nil, err
	}
	return peer, nil
}

func (p *PeerManager) GetLocalDataDecrypter() (cryptkit.Decrypter, error) {
	return p.Local().getDataDecrypter()
}

func (p *PeerManager) Manager() uniproto.PeerManager {
	return facadePeerManager{
		peerManager:    p,
		maxSessionless: p.central.maxSessionlessSize,
		maxSmall:       uniproto.MaxNonExcessiveLength,
	}
}

/****************************/

var _ uniproto.PeerManager = &facadePeerManager{}

type facadePeerManager struct {
	peerManager    *PeerManager
	maxSessionless uint16
	maxSmall       uint16
}

func (v facadePeerManager) ConnectPeer(address nwapi.Address) (uniproto.Peer, error) {
	switch peer, err := v.peerManager.connectPeer(address); {
	case peer != nil:
		return peer, err
	case err != nil:
		return nil, err
	default:
		return nil, throw.Impossible()
	}
}

func (v facadePeerManager) ConnectedPeer(address nwapi.Address) (uniproto.Peer, error) {
	switch peer, err := v.peerManager.peerNotLocal(address); {
	case err != nil:
		return nil, err
	case peer != nil:
		return peer, nil
	default:
		return nil, throw.Impossible()
	}
}

func (v facadePeerManager) LocalPeer() uniproto.Peer {
	return v.peerManager.Local()
}

func (v facadePeerManager) MaxSessionlessPayloadSize() uint {
	return uint(v.maxSessionless)
}

func (v facadePeerManager) MaxSmallPayloadSize() uint {
	return uint(v.maxSmall)
}
