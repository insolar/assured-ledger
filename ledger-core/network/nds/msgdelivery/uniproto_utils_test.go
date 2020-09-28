// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func NewUnitProtoServersHolder(log uniserver.MiniLogger) *UnitProtoServersHolder {
	return &UnitProtoServersHolder{
		servers:            make([]*UnitProtoServer, 0),
		log:                log,
		initialPulseNumber: pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{})),
	}
}

type UnitProtoServersHolder struct {
	servers            []*UnitProtoServer
	log                uniserver.MiniLogger
	initialPulseNumber pulse.Range
}

type UnitProtoServer struct {
	hostId     nwapi.HostID
	service    Service
	ingoing    nwapi.Address
	outgoing   sync.Map
	key        cryptkit.SigningKey
	dispatcher *uniserver.Dispatcher
	manager    *uniserver.PeerManager
}

func (s *UnitProtoServer) directAddress() DeliveryAddress {
	return NewDirectAddress(nwapi.ShortNodeID(s.hostId))
}

func (h *UnitProtoServersHolder) server(idx int) *UnitProtoServer {
	if idx < 1 || idx > len(h.servers) {
		panic("")
	}

	return h.servers[idx-1]
}

func (h *UnitProtoServersHolder) stop() {
	// TODO workaround for avoid race on receive package and stop dispatcher
	time.Sleep(1 * time.Second)
	for _, s := range h.servers {
		s.dispatcher.Stop()
	}
}

func (h *UnitProtoServersHolder) createService(
	config uniserver.ServerConfig,
	receiverFn ReceiverFunc,
) (*UnitProtoServer, error) {
	return h.createServiceWithProfile(NewUnitProtoServerProfile(config), receiverFn)
}

func (h *UnitProtoServersHolder) createServiceWithProfile(
	profile *UnitProtoServerProfile,
	receiverFn ReceiverFunc,
) (*UnitProtoServer, error) {
	controller := NewController(Protocol, profile.getDesFactory(), receiverFn, profile.getResolverFn(), h.log)

	var dispatcher uniserver.Dispatcher
	dispatcher.SetMode(uniproto.AllowAll)
	controller.RegisterWith(dispatcher.RegisterProtocol)
	dispatcher.Seal()

	srv := uniserver.NewUnifiedServer(&dispatcher, h.log)
	srv.SetConfig(profile.getConfig())

	// add self hostId mapping
	hostId := nwapi.HostID(len(h.servers) + 1)
	serv := &UnitProtoServer{
		hostId:     hostId,
		service:    controller.NewFacade(),
		ingoing:    nwapi.Address{},
		outgoing:   sync.Map{},
		key:        newSkKey(),
		dispatcher: &dispatcher,
	}
	h.servers = append(h.servers, serv)

	peerFn := func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		reg := func(idx int, s *UnitProtoServer) (nwapi.Address, error) {
			id := idx + 1
			peer.SetNodeID(nwapi.ShortNodeID(id))
			peer.SetSignatureKey(s.key)

			return nwapi.NewHostID(nwapi.HostID(id)), nil
		}

		var addr *nwapi.Address

	outer:
		for {
			for idx, s := range h.servers {
				s.outgoing.Range(func(key, value interface{}) bool {
					key0 := key.(nwapi.Address)
					if key0 == peer.GetPrimary() {
						add0, _ := reg(idx, s)
						addr = &add0
						return false
					}
					return true
				})

				if addr != nil {
					break outer
				}
				if addr == nil {
					if s.ingoing == peer.GetPrimary() {
						add0, _ := reg(idx, s)
						addr = &add0
						break outer
					}
				}
			}
		}

		return *addr, nil
	}

	// provider for intercept outgoing connections
	provider := uniserver.MapTransportProvider(&uniserver.DefaultTransportProvider{},
		func(lessProvider l1.SessionlessTransportProvider) l1.SessionlessTransportProvider {
			return lessProvider
		},
		func(fullProvider l1.SessionfulTransportProvider) l1.SessionfulTransportProvider {
			return l1.MapSessionFullProvider(fullProvider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return regAddrOutTransportFactory{
					OutTransportFactory: factory,
					regAddr: func(addr nwapi.Address) {
						// save all outgoing addresses for future matching address -> hostId
						serv.outgoing.Store(addr, nil)
					},
				}
			})
		})

	srv.SetTransportProvider(provider)

	srv.SetPeerFactory(peerFn)
	srv.SetSignatureFactory(TestVerifierFactory{})

	srv.StartListen()
	dispatcher.NextPulse(h.initialPulseNumber)

	manager := srv.PeerManager()
	primaryAddr := manager.Local().GetPrimary()
	// add self hostId
	_, err := manager.AddHostID(primaryAddr, hostId)
	if err != nil {
		return nil, err
	}

	serv.manager = manager
	serv.ingoing = primaryAddr

	for _, s := range h.servers {
		if s == serv {
			break
		}
		_, err = manager.Manager().ConnectPeer(s.manager.Local().GetPrimary())
		if err != nil {
			return nil, err
		}
		_, err := s.manager.Manager().ConnectPeer(primaryAddr)
		if err != nil {
			return nil, err
		}
	}

	return serv, nil
}

func newSkKey() cryptkit.SigningKey {
	skBytes := [testDigestSize]byte{}
	skBytes[0] = 1
	skKey := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)
	return skKey
}

type regAddrOutTransportFactory struct {
	l1.OutTransportFactory
	regAddr func(nwapi.Address)
}

func (r regAddrOutTransportFactory) ConnectTo(address nwapi.Address) (l1.OneWayTransport, error) {
	t, err := r.OutTransportFactory.ConnectTo(address)

	if t0, ok := t.(l1.OutNetTransport); ok {
		localAddr := t0.NetConn().LocalAddr()

		r.regAddr(nwapi.AsAddress(localAddr))
	}

	return t, err
}

type UnitProtoServerProfile struct {
	config       uniserver.ServerConfig
	provider     uniserver.AllTransportProvider
	desFactory   nwapi.DeserializationFactory
	idWithPortFn func(nwapi.Address) bool
	resolverFn   ResolverFunc
}

func NewUnitProtoServerProfile(config uniserver.ServerConfig) *UnitProtoServerProfile {
	return &UnitProtoServerProfile{config: config}
}

func (p *UnitProtoServerProfile) getConfig() uniserver.ServerConfig {
	return p.config
}

func (p *UnitProtoServerProfile) getProvider() uniserver.AllTransportProvider {
	return p.provider
}

func (p *UnitProtoServerProfile) getDesFactory() nwapi.DeserializationFactory {
	if p.desFactory != nil {
		return p.desFactory
	}

	return &TestDeserializationFactory{}
}

func (p *UnitProtoServerProfile) getIdWithPortFn() func(nwapi.Address) bool {
	return p.idWithPortFn
}

func (p *UnitProtoServerProfile) getResolverFn() ResolverFunc {
	return p.resolverFn
}
