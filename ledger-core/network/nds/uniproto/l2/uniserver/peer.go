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
	//Established
	Connected
	Verified
	Trusted
)

type Peer struct {
	transport PeerTransport

	pk  cryptkit.SignatureKey
	dsv cryptkit.DataSignatureVerifier
	dsg cryptkit.DataSigner

	nodeID nwapi.ShortNodeID
	state  atomickit.Uint32 // PeerState

	// HostIds for indirectly accessible hosts?

	protoMutex sync.RWMutex
	protoInfo  [uniproto.ProtocolTypeCount]io.Closer
}

func (p *Peer) GetPrimary() nwapi.Address {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	if aliases := p.transport.aliases; len(aliases) > 0 {
		return aliases[0]
	}
	return nwapi.Address{}
}

func (p *Peer) updatePrimary(addr nwapi.Address) nwapi.Address {
	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	switch aliases := p.transport.aliases; {
	case len(aliases) == 0:
		//
	case aliases[0] == addr:
		//
	default:
		prev := aliases[0]
		aliases[0] = addr
		return prev
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

func (p *Peer) verifyByTLS(_ *tls.Conn) (verified bool, err error) {
	return false, nil
}

func (p *Peer) SetSignatureKey(pk cryptkit.SignatureKey) {
	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	if p.pk.Equals(pk) {
		return
	}

	p.pk = pk
	p.dsv = nil
	p.dsg = nil
}

func (p *Peer) GetSignatureKey() cryptkit.SignatureKey {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

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

func (p *Peer) GetDataVerifier() (cryptkit.DataSignatureVerifier, error) {
	p.transport.mutex.RLock()
	if d := p.dsv; d != nil {
		p.transport.mutex.RUnlock()
		return d, nil
	}
	p.transport.mutex.RUnlock()

	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	if d := p.dsv; d != nil {
		return d, nil
	}
	err := p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		p.dsv = f.CreateDataSignatureVerifier(p.pk)
		return p.dsv != nil
	})
	return p.dsv, err
}

func (p *Peer) GetDataSigner() (cryptkit.DataSigner, error) {
	p.transport.mutex.RLock()
	if d := p.dsg; d != nil {
		p.transport.mutex.RUnlock()
		return d, nil
	}
	p.transport.mutex.RUnlock()

	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	if d := p.dsg; d != nil {
		return d, nil
	}
	err := p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		p.dsg = f.CreateDataSigner(p.pk)
		return p.dsg != nil
	})
	return p.dsg, err
}

func (p *Peer) getDataDecrypter() (d cryptkit.Decrypter, err error) {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	err = p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		d = f.CreateDataDecrypter(p.pk)
		return d != nil
	})
	return d, err
}

func (p *Peer) GetDataEncrypter() (d cryptkit.Encrypter, err error) {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	err = p._useKeyAndFactory(func(f PeerCryptographyFactory) bool {
		d = f.CreateDataEncrypter(p.pk)
		return d != nil
	})
	return d, err
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
	if id == 0 {
		panic(throw.IllegalState())
	}

	p.transport.mutex.Lock()
	defer p.transport.mutex.Unlock()

	if !p.nodeID.IsAbsent() {
		panic(throw.IllegalState())
	}
	p.nodeID = id
}

func (p *Peer) GetLocalUID() nwapi.Address {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	return nwapi.NewLocalUID(p.transport.uid, nwapi.HostID(p.nodeID))
}

func (p *Peer) GetHostID() nwapi.HostID {
	p.transport.mutex.RLock()
	defer p.transport.mutex.RUnlock()

	return nwapi.HostID(p.nodeID)
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
	p.protoMutex.Lock()
	defer p.protoMutex.Unlock()
	if r := p.protoInfo[pt]; r != nil {
		return r
	}
	r := factoryFn(p)
	p.protoInfo[pt] = r
	return r
}

func (p *Peer) GetProtoInfo(pt uniproto.ProtocolType) io.Closer {
	p.protoMutex.RLock()
	defer p.protoMutex.RUnlock()
	return p.protoInfo[pt]
}

func (p *Peer) Transport() uniproto.OutTransport {
	return &p.transport
}

func (p *Peer) SendPacket(tp uniproto.OutType, packet uniproto.PacketPreparer) error {
	t, sz, fn := packet.PreparePacket()
	return p.SendPreparedPacket(tp, &t.Packet, sz, fn, nil)
}

func (p *Peer) SendPreparedPacket(tp uniproto.OutType, packet *uniproto.Packet, dataSize uint, fn uniproto.PayloadSerializerFunc, checkFn func() bool) error {

	sp := p.createSendingPacket(packet)

	packetSize, sendFn := sp.NewTransportFunc(dataSize, fn, checkFn)
	switch tp {
	case uniproto.Any:
		if packetSize <= uint(p.transport.central.maxSessionlessSize) {
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
		if packetSize <= uint(p.transport.central.maxSessionlessSize) {
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
		if packetSize > uint(p.transport.central.maxSessionlessSize) {
			panic(throw.FailHere("too big for sessionless packet"))
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
	if sig, err = p.GetDataSigner(); err != nil {
		panic(err)
	}
	sp := uniproto.NewSendingPacket(sig, enc)
	sp.Packet = *packet
	sp.Peer = p
	return sp
}
