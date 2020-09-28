// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"crypto/tls"
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type PeerState uint8

const (
	_ PeerState = iota
	// Established
	Connected
	Verified
	Trusted
)

// Peer represents a logical peer for communications. One Peer may have multiple addresses and identities.
// Initially, a Peer is created by connection's remote address, but a Peer can be merged with another Peer after identification.
// Also, Peer can have multiple connections of different types.
type Peer struct {
	transport PeerTransport

	// LOCK: WARNING! mutex can be acquired under PeerManager.peerMutex
	// mutex controls local attributes
	mutex sync.RWMutex

	pk  cryptkit.SigningKey
	dsv cryptkit.DataSignatureVerifier
	dsg cryptkit.DataSigner

	nodeID nwapi.ShortNodeID
	state  atomickit.Uint32 // PeerState

	// HostIds for indirectly accessible hosts?

	protoMutex sync.RWMutex
	// protoInfo contains links to Peer's projection for each protocol it was connected through.
	protoInfo [uniproto.ProtocolTypeCount]io.Closer
}

// GetPrimary returns a peer's primary identity
func (p *Peer) GetPrimary() nwapi.Address {
	return p.transport.getPrimary()
}

// updatePrimary is ONLY for use by server to update Local peer identity after start of listening.
func (p *Peer) updatePrimary(addr nwapi.Address) nwapi.Address {
	return p.transport.updatePrimary(addr)
}

// LOCK: WARNING! runs under PeerManager.peerMutex
func (p *Peer) onRemoved() []nwapi.Address {
	p.mutex.Lock()
	info := p.protoInfo
	p.protoInfo = [uniproto.ProtocolTypeCount]io.Closer{}
	p.mutex.Unlock()

	for _, c := range info {
		_ = iokit.SafeClose(c)
	}

	return p.transport.kill()
}

// verifyByTLS is invoked to verify TLS connection.
// Return (true, nil) for positive on peer's identity (accept).
// Return (false, nil) for unknown, neither positive nor negative (proceed further).
// Return (-, err) for negative on peer's identity (deny).
// NB! this can use a better abstraction.
func (p *Peer) verifyByTLS(_ *tls.Conn) (verified bool, err error) {
	return false, nil
}

// SetSignatureKey sets/updates peer's signature key. Key can be zero.
func (p *Peer) SetSignatureKey(pk cryptkit.SigningKey) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.pk.Equals(pk) {
		return
	}

	p.pk = pk
	p.dsv = nil
	p.dsg = nil
}

// GetSignatureKey returns peer's signature key. Key can be zero.
func (p *Peer) GetSignatureKey() cryptkit.SigningKey {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.pk
}

func (p *Peer) _useKeyAndFactory(fn func(PeerCryptographyFactory) bool) error {
	switch f := p.transport.central.sigFactory; {
	case f == nil:
		return throw.IllegalState()
	case p.pk.IsEmpty():
		return throw.FailCaller("key is missing", 1)
	case !f.IsSignatureKeySupported(p.pk):
		return throw.FailCaller("key is unsupported", 1)
	case !fn(f):
		return throw.FailCaller("factory has failed", 1)
	}
	return nil
}

// GetDataVerifier returns DataSignatureVerifier to check data provided by this peer. Will return error when SignatureKey is not set.
func (p *Peer) GetDataVerifier() (cryptkit.DataSignatureVerifier, error) {
	p.mutex.RLock()
	if d := p.dsv; d != nil {
		p.mutex.RUnlock()
		return d, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if d := p.dsv; d != nil {
		return d, nil
	}
	err := p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		p.dsv = f.CreateDataSignatureVerifier(p.pk)
		return p.dsv != nil
	})
	return p.dsv, err
}

// getDataSigner returns DataSigner to sign data by this peer. Applicable to Local peer only.
// Will return error when SignatureKey is not set.
func (p *Peer) getDataSigner() (cryptkit.DataSigner, error) {
	p.mutex.RLock()
	if d := p.dsg; d != nil {
		p.mutex.RUnlock()
		return d, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if d := p.dsg; d != nil {
		return d, nil
	}
	err := p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		p.dsg = f.CreateDataSigner(p.pk)
		return p.dsg != nil
	})
	return p.dsg, err
}

// getDataSigner returns Decrypter to decrypt data for this peer. Applicable to Local peer only.
// Will return error when SignatureKey is not set.
func (p *Peer) getDataDecrypter() (d cryptkit.Decrypter, err error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	err = p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		d = f.CreateDataDecrypter(p.pk)
		return d != nil
	})
	return d, err
}

// GetDataVerifier returns Encrypter to encrypt data for this peer. Will return error when SignatureKey is not set.
func (p *Peer) GetDataEncrypter() (d cryptkit.Encrypter, err error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	err = p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		d = f.CreateDataEncrypter(p.pk)
		return d != nil
	})
	return d, err
}

// UpgradeState rises state of this peer. Will ignore lower values.
func (p *Peer) UpgradeState(state PeerState) {
	p.state.SetGreater(uint32(state))
}

func (p *Peer) getState() PeerState {
	return PeerState(p.state.Load())
}

// HasState returns true when the peer has same or higher state.
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

// GetNodeID returns ShortNodeID of the peer. Result can be zero. ShortNodeID is related to consensus operations.
func (p *Peer) GetNodeID() nwapi.ShortNodeID {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.nodeID
}

// SetNodeID sets peer's ShortNodeID. Will panic on zero.
func (p *Peer) SetNodeID(id nwapi.ShortNodeID) {
	if id == 0 {
		panic(throw.IllegalState())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.nodeID.IsAbsent() {
		panic(throw.IllegalState())
	}
	p.nodeID = id
}

// GetLocalUID returns a locally unique peer's address.
// This address can only be used locally and it will change after each disconnection of the peer.
func (p *Peer) GetLocalUID() nwapi.Address {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return nwapi.NewLocalUID(p.transport.uid, nwapi.HostID(p.nodeID))
}

// SetProtoInfo sets/updated per-protocol peer's projection (info). It will be closed when the peer is unregistered.
func (p *Peer) SetProtoInfo(pt uniproto.ProtocolType, info io.Closer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.protoInfo[pt] = info
}

// GetOrCreateProtoInfo performs "atomic" get or create of peer's projection for the given protocol.
func (p *Peer) GetOrCreateProtoInfo(pt uniproto.ProtocolType, factoryFn func(uniproto.Peer) io.Closer) io.Closer {
	if factoryFn == nil {
		panic(throw.IllegalValue())
	}

	if r := p.GetProtoInfo(pt); r != nil {
		return r
	}
	p.protoMutex.Lock()
	defer p.protoMutex.Unlock()
	if r := p.protoInfo[pt]; r != nil {
		return r
	}
	r := factoryFn(p)
	p.protoInfo[pt] = r
	return r
}

// GetProtoInfo returns peer's projection for the given protocol. Result can be nil.
func (p *Peer) GetProtoInfo(pt uniproto.ProtocolType) io.Closer {
	p.protoMutex.RLock()
	defer p.protoMutex.RUnlock()
	return p.protoInfo[pt]
}

// Transport returns a multiplexed transport over available peer's transports.
// This transport does internal connection management (retry, reconnect, etc).
func (p *Peer) Transport() uniproto.OutTransport {
	return &p.transport
}

// SendPacket is a convenience handler to send a packet provided by uniproto.PacketPreparer.
// See SendPreparedPacket for details.
func (p *Peer) SendPacket(tp uniproto.OutType, packet uniproto.PacketPreparer) error {
	t, sz, fn := packet.PreparePacket()
	return p.SendPreparedPacket(tp, &t.Packet, sz, fn, nil)
}

// SendPreparedPacket prepares uniproto packet and checks its eligibility for the given (uniproto.OutType).
// Can also choose a specific out type by properties of packet and value of (uniproto.OutType).
// See SendingPacket.NewTransportFunc for details about serialization.
func (p *Peer) SendPreparedPacket(tp uniproto.OutType, packet *uniproto.Packet, dataSize uint, fn uniproto.PayloadSerializerFunc, checkFn func() bool) error {
	sp := p.createSendingPacket(packet)

	packetSize, sendFn := sp.NewTransportFunc(dataSize, fn, checkFn)
	switch tp {
	case uniproto.Any:
		if p.transport.central.isSessionlessAllowed(int(packetSize)) {
			tp = uniproto.Sessionless
			break
		}
		fallthrough
	case uniproto.SessionfulAny:
		if packetSize <= uniproto.MaxNonExcessiveLength {
			tp = uniproto.SessionfulSmall
		} else {
			tp = uniproto.SessionfulLarge
		}
	case uniproto.SmallAny:
		if p.transport.central.isSessionlessAllowed(int(packetSize)) {
			tp = uniproto.Sessionless
			break
		}
		tp = uniproto.SessionfulSmall
		fallthrough
	case uniproto.SessionfulSmall:
		if packetSize > uniproto.MaxNonExcessiveLength {
			panic(throw.FailHere("too big for non-excessive packet"))
		}
	case uniproto.Sessionless, uniproto.SessionlessNoQuota:
		if !p.transport.central.isSessionlessAllowed(int(packetSize)) {
			panic(throw.FailHere("sessionless is not allowed"))
		}
	}
	return p.transport.sendPacket(tp, sendFn)
}

func (p *Peer) createSendingPacket(packet *uniproto.Packet) *uniproto.SendingPacket {
	var (
		sig cryptkit.DataSigner
		enc cryptkit.Encrypter
		err error
	)

	if packet.Header.IsBodyEncrypted() {
		if enc, err = p.GetDataEncrypter(); err != nil {
			panic(err)
		}
	}
	if sig, err = p.getDataSigner(); err != nil {
		panic(err)
	}
	sp := uniproto.NewSendingPacket(sig, enc)
	sp.Packet = *packet
	sp.Peer = p
	return sp
}
