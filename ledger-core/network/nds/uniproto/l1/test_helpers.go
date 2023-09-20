package l1

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type FuncMapOutputFactory func(OutTransportFactory) OutTransportFactory
type FuncMapOneWayTransport func(OneWayTransport) OneWayTransport

func MapSessionlessProvider(
	provider SessionlessTransportProvider,
	mappingFunc FuncMapOutputFactory,
) SessionlessTransportProvider {
	return &MappedSessionlessTransportProvider{
		SessionlessTransportProvider: provider,
		mappingFunc:                  mappingFunc,
	}
}

type MappedSessionlessTransportProvider struct {
	SessionlessTransportProvider

	mappingFunc func(OutTransportFactory) OutTransportFactory
}

func (m *MappedSessionlessTransportProvider) CreateListeningFactory(fun SessionlessReceiveFunc) (OutTransportFactory, nwapi.Address, error) {
	factory, address, err := m.SessionlessTransportProvider.CreateListeningFactory(fun)

	return m.mappingFunc(factory), address, err
}

func (m *MappedSessionlessTransportProvider) CreateListeningFactoryWithAddress(fun SessionlessReceiveFunc, addr nwapi.Address) (OutTransportFactory, error) {
	factory, err := m.SessionlessTransportProvider.CreateListeningFactoryWithAddress(fun, addr)

	return m.mappingFunc(factory), err
}

func (m *MappedSessionlessTransportProvider) CreateOutgoingOnlyFactory() (OutTransportFactory, error) {
	factory, err := m.SessionlessTransportProvider.CreateOutgoingOnlyFactory()

	return m.mappingFunc(factory), err
}

////////////////////////////////////

func MapSessionFullProvider(
	provider SessionfulTransportProvider,
	mappingFunc FuncMapOutputFactory,
) SessionfulTransportProvider {
	return &MappedSessionfullTransportProvider{
		SessionfulTransportProvider: provider,
		mappingFunc:                 mappingFunc,
	}
}

type MappedSessionfullTransportProvider struct {
	SessionfulTransportProvider

	mappingFunc func(OutTransportFactory) OutTransportFactory
}

func (w *MappedSessionfullTransportProvider) CreateListeningFactory(fun SessionfulConnectFunc) (OutTransportFactory, nwapi.Address, error) {
	factory, address, err := w.SessionfulTransportProvider.CreateListeningFactory(fun)

	return w.mappingFunc(factory), address, err
}

func (w *MappedSessionfullTransportProvider) CreateOutgoingOnlyFactory(fun SessionfulConnectFunc) (OutTransportFactory, error) {
	factory, err := w.SessionfulTransportProvider.CreateOutgoingOnlyFactory(fun)

	return w.mappingFunc(factory), err
}

///////////////////////////////////

func MapOutputFactory(factory OutTransportFactory, mappingFunc FuncMapOneWayTransport) OutTransportFactory {
	return &MappedOutputTransportFactory{
		OutTransportFactory: factory,
		mappingFunc:         mappingFunc,
	}
}

type MappedOutputTransportFactory struct {
	OutTransportFactory
	mappingFunc func(OneWayTransport) OneWayTransport
}

func (w *MappedOutputTransportFactory) ConnectTo(add nwapi.Address) (OneWayTransport, error) {
	con, err := w.OutTransportFactory.ConnectTo(add)

	return w.mappingFunc(con), err
}
