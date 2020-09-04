// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"context"
	"crypto/tls"
	"io"
	"math"
	"net"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type ServerConfig struct {
	BindingAddress string
	PublicAddress  string
	NetPreference  nwapi.Preference
	TLSConfig      *tls.Config
	UDPMaxSize     int
	UDPParallelism int
	PeerLimit      int

	// For sessionful connections only
	RetryLimit         uint8 // +1 attempt
	RetryDelayInc      time.Duration
	RetryDelayVariance time.Duration
	RetryDelayMax      time.Duration
}

type MiniLogger interface {
	LogError(error)
	LogTrace(interface{})
}

func NewUnifiedServer(dispatcher uniproto.Dispatcher, logger MiniLogger) *UnifiedServer {
	switch {
	case dispatcher == nil:
		panic(throw.IllegalValue())
	case logger == nil:
		panic(throw.IllegalValue())
	}
	s := &UnifiedServer{logger: logger}
	s.receiver.Parser.Dispatcher = dispatcher
	return s
}

type UnifiedServer struct {
	config    ServerConfig
	blacklist BlacklistManager
	logger    MiniLogger

	ptf        peerTransportFactory
	peers      PeerManager
	receiver   PeerReceiver
	udpSema    synckit.Semaphore
}

func (p *UnifiedServer) SetConfig(config ServerConfig) {
	p.config = config

	p.peers.central.maxSessionlessSize = uint16(config.UDPMaxSize)
	p.peers.central.retryLimit = config.RetryLimit
	p.peers.central.retryDelayInc = config.RetryDelayInc
	p.peers.central.retryDelayMax = config.RetryDelayMax
	p.peers.central.retryDelayVariance = config.RetryDelayVariance
}

func (p *UnifiedServer) SetQuotaFactory(quotaFn PeerQuotaFactoryFunc) {
	p.peers.SetQuotaFactory(quotaFn)
}

func (p *UnifiedServer) SetPeerFactory(fn PeerMapperFunc) {
	p.peers.SetPeerMapper(fn)
}

func (p *UnifiedServer) SetSignatureFactory(f PeerCryptographyFactory) {
	p.peers.SetSignatureFactory(f)
}

func (p *UnifiedServer) SetBlacklistManager(blacklist BlacklistManager) {
	p.blacklist = blacklist
}

func (p *UnifiedServer) SetRelayer(r Relayer) {
	p.receiver.Relayer = r
}

func (p *UnifiedServer) SetHTTPReceiver(fn HTTPReceiverFunc) {
	p.receiver.HTTP = fn
}

func (p *UnifiedServer) StartNoListen() {
	if p.peers.central.factory != nil {
		panic(throw.IllegalState())
	}

	p.receiver.Parser.SigSizeHint = p.peers.central.sigFactory.GetMaxSignatureSize()
	if p.receiver.Parser.SigSizeHint <= 0 {
		panic(throw.IllegalState())
	}

	if p.config.UDPParallelism > 0 {
		p.udpSema = synckit.NewSemaphore(p.config.UDPParallelism)
	} else {
		p.udpSema = synckit.NewSemaphore(4)
	}

	p.peers.central.factory = &p.ptf
	switch n := p.config.PeerLimit; {
	case n < 0:
		p.peers.central.maxPeerConn = 4
	case n >= math.MaxUint8:
		p.peers.central.maxPeerConn = math.MaxUint8
	default:
		p.peers.central.maxPeerConn = uint8(n)
	}

	binding := nwapi.NewHostPort(p.config.BindingAddress, true)
	localAddrs, _, err := nwapi.ExpandHostAddresses(context.Background(), false, net.DefaultResolver, binding)
	if err != nil {
		panic(err)
	}
	binding = p.config.NetPreference.ChooseOne(localAddrs)

	public := binding
	switch {
	case p.config.PublicAddress != "":
		public = nwapi.NewHostPort(p.config.PublicAddress, false)

		pubAddrs, _, err := nwapi.ExpandHostAddresses(context.Background(), false, net.DefaultResolver, public)
		if err != nil {
			panic(err)
		}
		localAddrs = nwapi.Join(localAddrs, pubAddrs)
	case !binding.HasPort():
		p.ptf.updateLocalAddr = p.peers.updateLocalPrimary
	}

	if err := p.peers.addLocal(public, localAddrs, func(peer *Peer) error {
		// TODO setup PrivateKey for signing
		return nil
	}); err != nil {
		panic(err)
	}

	udpSize := p.config.UDPMaxSize
	switch {
	case udpSize < 0:
		udpSize = l1.MaxUDPSize
	case udpSize > math.MaxUint16:
		udpSize = math.MaxUint16
	}
	p.ptf.SetSessionless(l1.NewUDP(binding, p.config.NetPreference, uint16(udpSize)), p.receiveSessionless)

	var tcp l1.SessionfulTransportProvider
	if p.config.TLSConfig == nil {
		tcp = l1.NewTCP(binding, p.config.NetPreference)
	} else {
		tcp = l1.NewTLS(binding, p.config.NetPreference, p.config.TLSConfig)
	}
	p.ptf.SetSessionful(tcp, p.connectSessionful, p.connectSessionfulListen)

	p.receiver.PeerManager = &p.peers
	d := p.receiver.Parser.Dispatcher
	p.receiver.Parser.Protocols = d.Seal()
	p.receiver.Parser.Dispatcher.Start(p.peers.Manager())
}

func (p *UnifiedServer) StartListen() {
	if !p.ptf.HasTransports() {
		p.StartNoListen()
	}

	if err := p.ptf.Listen(); err != nil {
		panic(err)
	}
}

func (p *UnifiedServer) PeerManager() *PeerManager {
	return &p.peers
}

func (p *UnifiedServer) Stop() {
	_ = p.ptf.Close()
	_ = p.peers.Close()
}

func (p *UnifiedServer) checkConnection(_, remote nwapi.Address, err error) error {
	switch {
	case err != nil:
		return err
	case p.isBlacklisted(remote):
		return throw.RemoteBreach("blacklisted")
	}
	return nil
}

func (p *UnifiedServer) receiveSessionless(local, remote nwapi.Address, b []byte, err error) bool {
	if !p.ptf.listen.IsActive() {
		return true
	}

	if p.udpSema.LockTimeout(time.Second) {
		b = append([]byte(nil), b...)

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
		p.reportError(throw.E("inbound packet dropped by timeout", ConnErrDetails{local, remote}))
	}

	return true
}

func (p *UnifiedServer) connectSessionfulListen(local, remote nwapi.Address, conn io.ReadWriteCloser, w l1.OneWayTransport, err error) bool {
	if !p.ptf.listen.IsActive() {
		// can't accept incoming connections until listen initializer is finished
		_ = conn.Close()
		return true
	}
	return p.connectSessionful(local, remote, conn, w, err)
}

func (p *UnifiedServer) connectSessionful(local, remote nwapi.Address, conn io.ReadWriteCloser, w l1.OneWayTransport, err error) bool {
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

func (p *UnifiedServer) runReceiver(local, remote nwapi.Address, runFn func() error) {
	if err := runFn(); err != nil {
		p.reportToBlacklist(remote, err)
		p.reportError(throw.WithDetails(err, ConnErrDetails{local, remote}))
	}
}

func (p *UnifiedServer) isBlacklisted(remote nwapi.Address) bool {
	return p.blacklist != nil && p.blacklist.IsBlacklisted(remote)
}

func (p *UnifiedServer) reportToBlacklist(remote nwapi.Address, err error) {
	if bl := p.blacklist; bl != nil {
		if sv := throw.SeverityOf(err); sv.IsFraudOrWorse() {
			bl.ReportFraud(remote, &p.peers, err)
		}
	}
}

func (p *UnifiedServer) reportError(err error) {
	p.logger.LogError(err)
}
