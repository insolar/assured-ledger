// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"crypto/tls"
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

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
	nodeID    nwapi.ShortNodeID
	state     atomickit.Uint32 // PeerState

	// HostIds for indirectly accessible hosts?

	protoInfo [uniproto.ProtocolTypeCount]io.Closer
}

func (p *Peer) GetPrimary() nwapi.Address {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	if aliases := p.transport.aliases; len(aliases) > 0 {
		return aliases[0]
	}
	return nwapi.Address{}
}

func (p *Peer) onRemoved() []nwapi.Address {
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

func (p *Peer) removeAlias(a nwapi.Address) bool {
	return p.transport.removeAlias(a)
}

func (p *Peer) GetNodeID() nwapi.ShortNodeID {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	return p.nodeID
}

func (p *Peer) SetNodeID(id nwapi.ShortNodeID) {
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

func (p *Peer) Transport() uniproto.OutTransport {
	return &p.transport
}
