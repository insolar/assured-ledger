// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"crypto/tls"
	"io"
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewPeerManager(factory PeerTransportFactory, local apinetwork.Address, localFn func(*Peer)) *PeerManager {
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
}

type PeerManager struct {
	quotaFactory PeerQuotaFactoryFunc
	sigFactory   PeerCryptographyFactory
	peerFactory  OfflinePeerFactoryFunc

	central PeerTransportCentral
	// LOCK: WARNING! PeerTransport.mutex can be acquired under this mutex
	peerMutex sync.RWMutex
	peers     PeerMap
}

type PeerQuotaFactoryFunc = func([]apinetwork.Address) ratelimiter.RWRateQuota
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
	p.sigFactory = f
}

func (p *PeerManager) peer(a apinetwork.Address) (uint32, *Peer) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p.peers.get(a)
}

func (p *PeerManager) peerNotLocal(a apinetwork.Address) (*Peer, error) {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()

	return p._peerNotLocal(a)
}

func (p *PeerManager) _peerNotLocal(a apinetwork.Address) (*Peer, error) {
	if idx, peer := p.peers.get(a); idx == 0 && peer != nil {
		return nil, throw.Violation("loopback is not allowed")
	} else {
		return peer, nil
	}
}

func (p *PeerManager) HasAddress(a apinetwork.Address) bool {
	return !p.GetPrimary(a).IsZero()
}

func (p *PeerManager) Local() *Peer {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()
	return p.peers.peers[0]
}

func (p *PeerManager) GetPrimary(a apinetwork.Address) apinetwork.Address {
	if _, peer := p.peer(a); peer != nil {
		return peer.GetPrimary()
	}
	return apinetwork.Address{}
}

func (p *PeerManager) RemovePeer(a apinetwork.Address) bool {
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

func (p *PeerManager) RemoveAlias(a apinetwork.Address) bool {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	return p.peers.removeAlias(a)
}

func (p *PeerManager) addLocal(primary apinetwork.Address, aliases []apinetwork.Address, newPeerFn func(*Peer) error) error {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	p.peers.ensureEmpty()
	if primary.IsZero() {
		panic(throw.IllegalValue())
	}

	_, _, err := p._newPeer(newPeerFn, primary, aliases)
	return err
}

func (p *PeerManager) AddPeer(primary apinetwork.Address, aliases ...apinetwork.Address) {
	if err := p.addPeer(primary, aliases...); err != nil {
		panic(err)
	}
}

func (p *PeerManager) addPeer(primary apinetwork.Address, aliases ...apinetwork.Address) error {
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

func (p *PeerManager) AddAliases(to apinetwork.Address, aliases ...apinetwork.Address) {
	if err := p.addAliases(to, aliases...); err != nil {
		panic(err)
	}
}

func (p *PeerManager) getPeer(a apinetwork.Address) (uint32, *Peer, error) {
	peerIndex, peer := p.peer(a)
	if peer == nil {
		return 0, nil, throw.E("unknown peer", struct{ apinetwork.Address }{a})
	}
	return peerIndex, peer, nil
}

func (p *PeerManager) addAliases(to apinetwork.Address, aliases ...apinetwork.Address) error {
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

func (p *PeerManager) _newPeer(newPeerFn func(*Peer) error, primary apinetwork.Address, aliases []apinetwork.Address) (uint32, *Peer, error) {
	peer := &Peer{}
	peer.transport.central = &p.central
	peer.transport.setAddresses(primary, aliases)

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

	if _, err := p.peers.checkAliases(nil, math.MaxUint32, peer.transport.aliases); err != nil {
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

func (p *PeerManager) connectionFrom(remote apinetwork.Address, newPeerFn func(*Peer) error) (*Peer, error) {
	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	peer, err := p._peerNotLocal(remote)
	if err == nil && peer == nil {
		_, peer, err = p._newPeer(newPeerFn, remote, nil)
	}
	return peer, err
}

func (p *PeerManager) AddHostId(to apinetwork.Address, id apinetwork.HostId) (bool, error) {
	peerIndex, peer, err := p.getPeer(to)
	if err != nil {
		return false, err
	}
	if id == 0 {
		return false, nil
	}

	if id.IsNodeId() {
		peer.SetNodeID(id.AsNodeId())
	}

	p.peerMutex.Lock()
	defer p.peerMutex.Unlock()

	err = p.peers.addAliases(peerIndex, []apinetwork.Address{apinetwork.NewHostId(id)})
	return err == nil, err
}

func (p *PeerManager) ConnectTo(remote apinetwork.Address) (uniproto.OutTransport, error) {
	switch peer, err := p.peerNotLocal(remote); {
	case err != nil:
		return nil, err
	case peer != nil:
		return &peer.transport, nil
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
	return &peer.transport, nil
}

func (p *PeerManager) GetDecrypter(*Peer) cryptkit.Decrypter {
	sk := p.Local().GetSignatureKey()
	return p.sigFactory.CreateDataDecrypter(sk)
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
	nodeID    apinetwork.ShortNodeID
	state     atomickit.Uint32 // PeerState

	// HostIds for indirectly accessible hosts?

	protoInfo [uniproto.ProtocolTypeCount]io.Closer
}

func (p *Peer) GetPrimary() apinetwork.Address {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	if aliases := p.transport.aliases; len(aliases) > 0 {
		return aliases[0]
	}
	return apinetwork.Address{}
}

func (p *Peer) onRemoved() []apinetwork.Address {
	info := p.protoInfo
	p.protoInfo = [uniproto.ProtocolTypeCount]io.Closer{}

	for _, c := range info {
		_ = iokit.SafeClose(c)
	}

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

func (p *Peer) GetSignatureKey() cryptkit.SignatureKey {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	return p.pk
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

func (p *Peer) removeAlias(a apinetwork.Address) bool {
	return p.transport.removeAlias(a)
}

func (p *Peer) GetNodeID() apinetwork.ShortNodeID {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	return p.nodeID
}

func (p *Peer) SetNodeID(id apinetwork.ShortNodeID) {
	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	if p.nodeID == id {
		return
	}

	if !p.nodeID.IsAbsent() {
		panic(throw.IllegalState())
	}
	p.nodeID = id
}

func (p *Peer) SetProtoInfo(pt uniproto.ProtocolType, info io.Closer) {
	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()
	p.protoInfo[pt] = info
}

func (p *Peer) GetOrCreateProtoInfo(pt uniproto.ProtocolType, factoryFn func(uniproto.Peer) io.Closer) io.Closer {
	if factoryFn == nil {
		panic(throw.IllegalValue())
	}

	if r := p.GetProtoInfo(pt); r != nil {
		return r
	}
	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()
	if r := p.protoInfo[pt]; r != nil {
		return r
	}
	r := factoryFn(p)
	p.protoInfo[pt] = r
	return r
}

func (p *Peer) GetProtoInfo(pt uniproto.ProtocolType) io.Closer {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()
	return p.protoInfo[pt]
}
