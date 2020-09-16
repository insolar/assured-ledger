// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"crypto/tls"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type TestTransportProvider struct {
	// binding    nwapi.Address
	// preference nwapi.Preference
	// maxUDPSize uint16
	eventsChan chan string
}

func (tp *TestTransportProvider) CreateSessionlessProvider(binding nwapi.Address, preference nwapi.Preference, maxUDPSize uint16) SessionlessTransportProvider {
	return NewUDP(binding, preference, maxUDPSize)
}

func (tp *TestTransportProvider) CreateSessionfulProvider(binding nwapi.Address, preference nwapi.Preference, tlsCfg *tls.Config) SessionfulTransportProvider {
	return NewTCP(binding, preference)
}

func (tp *TestTransportProvider) events() <-chan string {
	return tp.eventsChan
}

// Uniserver.SetTransportProvider(provider AllTransportProvider)
// ======= TestTransportProvider -> tcpProvider  -> tcpTransportFactory - > TestTcpOutTransport

var _ TwoWayTransport = &tcpOutTransport{}
var _ OutNetTransport = &tcpOutTransport{}

type TestTcpOutTransport struct {
	events chan string
	tcpOutTransport
}

func (p *TestTcpOutTransport) Write(b []byte) (n int, err error) {
	// todo: intercept here
	p.events <- "Write"
	return p.tcpOutTransport.Write(b)
}

func (p *TestTcpOutTransport) Close() error {
	// todo: intercept here
	p.events <- "close"
	return p.tcpOutTransport.Close()
}

func (p *TestTcpOutTransport) ReadFrom(r io.Reader) (int64, error) {
	// todo: intercept here
	p.events <- "ReadFrom"
	return p.tcpOutTransport.ReadFrom(r)
}

func (p *TestTcpOutTransport) Send(payload io.WriterTo) error {
	// todo: intercept here
	p.events <- "Send"
	return p.tcpOutTransport.Send(payload)
}

func (p *TestTcpOutTransport) SendBytes(payload []byte) error {
	// todo: intercept here
	p.events <- "SendBytes"
	return p.tcpOutTransport.SendBytes(payload)
}
