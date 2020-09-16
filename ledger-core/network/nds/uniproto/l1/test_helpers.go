// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"io"
)

func WrapSessionLessProvider(wrapFunc func(OutTransportFactory) OutTransportFactory, provider SessionlessTransportProvider) SessionlessTransportProvider {
	return &WrapperSessionlessTransportProvider{
		SessionlessTransportProvider: provider,
		wrap:                         wrapFunc,
	}
}

type WrapperSessionlessTransportProvider struct {
	SessionlessTransportProvider

	wrap func(OutTransportFactory) OutTransportFactory
}

func (w *WrapperSessionlessTransportProvider) CreateListeningFactory(fun SessionlessReceiveFunc) (OutTransportFactory, nwapi.Address, error) {
	factory, address, err := w.SessionlessTransportProvider.CreateListeningFactory(fun)

	return w.wrap(factory), address, err
}

func (w *WrapperSessionlessTransportProvider) CreateListeningFactoryWithAddress(fun SessionlessReceiveFunc, addr nwapi.Address) (OutTransportFactory, error) {
	factory, err := w.SessionlessTransportProvider.CreateListeningFactoryWithAddress(fun, addr)

	return w.wrap(factory), err
}

func (w *WrapperSessionlessTransportProvider) CreateOutgoingOnlyFactory() (OutTransportFactory, error) {
	factory, err := w.SessionlessTransportProvider.CreateOutgoingOnlyFactory()

	return w.wrap(factory), err
}

////////////////////////////////////

func WrapOutPutFactory(wrap func(OneWayTransport) OneWayTransport, factory OutTransportFactory) OutTransportFactory {
	return &WrapperOutTransportFactory{
		OutTransportFactory: factory,
		wrap:                wrap,
	}
}

type WrapperOutTransportFactory struct {
	OutTransportFactory
	wrap func(OneWayTransport) OneWayTransport
}

func (w *WrapperOutTransportFactory) ConnectTo(add nwapi.Address) (OneWayTransport, error) {
	con, err := w.OutTransportFactory.ConnectTo(add)

	return w.wrap(con), err
}

////////////////////////////////////////

func WrapTransport(delegate OneWayTransport, t OneWayTransport) OneWayTransport {
	return &WrapperOneWayTransport{
		OneWayTransport: t,
		delegate:        delegate,
	}
}

type WrapperOneWayTransport struct {
	OneWayTransport
	delegate OneWayTransport
}

func (w *WrapperOneWayTransport) Send(payload io.WriterTo) error {
	if err := w.delegate.Send(payload); err != nil {
		return err
	}

	return w.OneWayTransport.Send(payload)
}

func (w *WrapperOneWayTransport) SendBytes(b []byte) error {
	if err := w.delegate.SendBytes(b); err != nil {
		return err
	}

	return w.OneWayTransport.SendBytes(b)
}
