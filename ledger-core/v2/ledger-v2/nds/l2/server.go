// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"context"
	"crypto/tls"
	"io"
	"math"
	"net"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ServerConfig struct {
	BindingAddress string
	PublicAddress  string
	NetPreference  apinetwork.NetworkPreference
	TlsConfig      *tls.Config
	UdpMaxSize     int
	PeerLimit      int
}

func NewUnifiedProtocolServer(protocols *apinetwork.UnifiedProtocolSet, updParallelism int) *UnifiedProtocolServer {
	if protocols == nil {
		panic(throw.IllegalValue())
	}
	if updParallelism <= 0 {
		updParallelism = 4
	}
	return &UnifiedProtocolServer{protocols: protocols, udpSema: synckit.NewSemaphore(updParallelism)}
}

type UnifiedProtocolServer struct {
	config    ServerConfig
	mode      atomickit.Uint32
	ptf       peerTransportFactory
	peers     PeerManager
	blacklist BlacklistManager

	protocols *apinetwork.UnifiedProtocolSet

	receiver PeerReceiver
	udpSema  synckit.Semaphore
}

func (p *UnifiedProtocolServer) SetConfig(config ServerConfig) {
	p.config = config
}

func (p *UnifiedProtocolServer) SetQuotaFactory(quotaFn PeerQuotaFactoryFunc) {
	p.peers.SetQuotaFactory(quotaFn)
}

func (p *UnifiedProtocolServer) SetPeerFactory(fn OfflinePeerFactoryFunc) {
	p.peers.SetPeerFactory(fn)
}

func (p *UnifiedProtocolServer) SetVerifierFactory(f cryptkit.DataSignatureVerifierFactory) {
	p.peers.SetVerifierFactory(f)
}

func (p *UnifiedProtocolServer) SetBlacklistManager(blacklist BlacklistManager) {
	p.blacklist = blacklist
}

func (p *UnifiedProtocolServer) StartNoListen() {

	p.peers.central.factory = &p.ptf
	switch n := p.config.PeerLimit; {
	case n < 0:
		p.peers.central.maxPeerConn = 4
	case n >= math.MaxUint8:
		p.peers.central.maxPeerConn = math.MaxUint8
	default:
		p.peers.central.maxPeerConn = uint8(n)
	}

	binding := apinetwork.NewHostPort(p.config.BindingAddress)
	localAddrs, _, err := apinetwork.ExpandHostAddresses(context.Background(), false, net.DefaultResolver, binding)
	if err != nil {
		panic(err)
	}
	binding = p.config.NetPreference.ChooseOne(localAddrs)

	public := binding
	if p.config.PublicAddress != "" {
		public = apinetwork.NewHostPort(p.config.PublicAddress)
		pubAddrs, _, err := apinetwork.ExpandHostAddresses(context.Background(), false, net.DefaultResolver, public)
		if err != nil {
			panic(err)
		}
		localAddrs = apinetwork.Join(localAddrs, pubAddrs)
	}

	if err := p.peers.addLocal(public, localAddrs, func(peer *Peer) error {
		// TODO setup PrivateKey for signing
		return nil
	}); err != nil {
		panic(err)
	}

	udpSize := p.config.UdpMaxSize
	switch {
	case udpSize < 0:
		udpSize = l1.MaxUdpSize
	case udpSize > math.MaxUint16:
		udpSize = math.MaxUint16
	}
	p.ptf.SetSessionless(l1.NewUdp(binding, uint16(udpSize)), p.receiveSessionless)

	if p.config.TlsConfig == nil {
		p.ptf.SetSessionful(l1.NewTcp(binding), p.connectSessionful)
	} else {
		p.ptf.SetSessionful(l1.NewTls(binding, *p.config.TlsConfig), p.connectSessionful)
	}

	p.receiver = PeerReceiver{&p.peers, p.protocols, p.GetMode}
}

func (p *UnifiedProtocolServer) StartListen() {
	if !p.ptf.HasTransports() {
		p.StartNoListen()
	}

	if err := p.ptf.Listen(); err != nil {
		panic(err)
	}
}

func (p *UnifiedProtocolServer) PeerManager() *PeerManager {
	return &p.peers
}

func (p *UnifiedProtocolServer) Stop() {
	_ = p.ptf.Close()
	_ = p.peers.Close()
}

func (p *UnifiedProtocolServer) GetMode() ConnectionMode {
	return ConnectionMode(p.mode.Load())
}

func (p *UnifiedProtocolServer) SetMode(mode ConnectionMode) {
	p.mode.Store(uint32(mode))
}

func (p *UnifiedProtocolServer) checkConnection(_, remote apinetwork.Address, err error) error {
	switch {
	case err != nil:
		return err
	case p.isBlacklisted(remote):
		return throw.RemoteBreach("blacklisted")
	}
	return nil
}

func (p *UnifiedProtocolServer) receiveSessionless(local, remote apinetwork.Address, b []byte, err error) bool {
	if p.udpSema.LockTimeout(time.Second) {
		go func() {
			defer p.udpSema.Unlock()

			// DO NOT report checkConnection errors to blacklist
			if err = p.checkConnection(local, remote, err); err == nil {
				if err = p.receiver.ReceiveDatagram(remote, b); err == nil {
					return
				}
				err = throw.WithDetails(err, ConnErrDetails{local, remote})
				p.reportToBlacklist(remote, err)
			} else {
				err = throw.WithDetails(err, ConnErrDetails{local, remote})
			}
			p.reportError(err)
		}()
	} else {
		p.reportError(throw.E("packet drop by timeout", ConnErrDetails{local, remote}))
	}

	return true
}

func (p *UnifiedProtocolServer) connectSessionful(local, remote apinetwork.Address, conn io.ReadWriteCloser, w l1.OutTransport, err error) bool {
	// DO NOT report checkConnection errors to blacklist
	if err = p.checkConnection(local, remote, err); err != nil {
		_ = conn.Close()
	} else if runFn, err2 := p.receiver.ReceiveStream(remote, conn, w); err2 == nil {
		go p.runReceiver(local, remote, runFn)
		return true
	} else {
		err = err2
	}

	p.reportError(throw.WithDetails(err, ConnErrDetails{local, remote}))
	return true
}

func (p *UnifiedProtocolServer) runReceiver(local, remote apinetwork.Address, runFn func() error) {
	if err := runFn(); err != nil {
		p.reportToBlacklist(remote, err)
		p.reportError(throw.WithDetails(err, ConnErrDetails{local, remote}))
	}
}

func (p *UnifiedProtocolServer) isBlacklisted(remote apinetwork.Address) bool {
	return p.blacklist != nil && p.blacklist.IsBlacklisted(remote)
}

func (p *UnifiedProtocolServer) reportToBlacklist(remote apinetwork.Address, err error) {
	if bl := p.blacklist; bl != nil {
		if sv := throw.SeverityOf(err); sv.IsFraudOrWorse() {
			bl.ReportFraud(remote, p.peers, err)
		}
	}
}

func (p *UnifiedProtocolServer) reportError(err error) {
	// TODO
	println()
	println(throw.ErrorWithStack(err))
}
