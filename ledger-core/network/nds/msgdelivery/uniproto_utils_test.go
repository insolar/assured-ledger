// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
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
	outgoing   []nwapi.Address
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
	controller := NewController(Protocol, TestDeserializationFactory{}, receiverFn, nil, h.log)

	var dispatcher uniserver.Dispatcher
	dispatcher.SetMode(uniproto.AllowAll)
	controller.RegisterWith(dispatcher.RegisterProtocol)
	dispatcher.Seal()

	srv := uniserver.NewUnifiedServer(&dispatcher, h.log)
	srv.SetConfig(config)

	// add self hostId mapping
	hostId := nwapi.HostID(len(h.servers) + 1)
	serv := &UnitProtoServer{
		outgoing: make([]nwapi.Address, 0),
	}

	peerFn := func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		reg := func(idx int, s *UnitProtoServer) (nwapi.Address, error) {
			id := idx + 1
			peer.SetNodeID(nwapi.ShortNodeID(id))
			peer.SetSignatureKey(s.key)
			return nwapi.NewHostID(nwapi.HostID(id)), nil
		}

		for idx, s := range h.servers {
			for _, a := range s.outgoing {
				if a == peer.GetPrimary() {
					return reg(idx, s)
				}
			}
			if s.ingoing == peer.GetPrimary() {
				return reg(idx, s)
			}
		}

		peer.SetSignatureKey(newSkKey())

		return nwapi.Address{}, nil
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
						serv.outgoing = append(serv.outgoing, addr)
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

	for _, s := range h.servers {
		_, err = manager.Manager().ConnectPeer(s.manager.Local().GetPrimary())
		if err != nil {
			return nil, err
		}
		_, err := s.manager.Manager().ConnectPeer(primaryAddr)
		if err != nil {
			return nil, err
		}
		_, err = s.manager.AddHostID(primaryAddr, hostId)
		if err != nil {
			return nil, err
		}
	}

	serv.service = controller.NewFacade()
	serv.key = newSkKey()
	serv.dispatcher = &dispatcher
	serv.manager = manager
	serv.ingoing = primaryAddr
	serv.hostId = hostId

	h.servers = append(h.servers, serv)
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
