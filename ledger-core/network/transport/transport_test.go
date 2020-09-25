// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package transport

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"

	"github.com/stretchr/testify/suite"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
)

type suiteTest struct {
	suite.Suite

	factory1 Factory
	factory2 Factory
}

type fakeNode struct {
	component.Starter
	component.Stopper

	tcp    StreamTransport
	tcpBuf chan []byte
}

func (f *fakeNode) HandleStream(ctx context.Context, address string, stream io.ReadWriteCloser) {
	inslogger.FromContext(ctx).Infof("HandleStream from %s", address)

	b := make([]byte, 3)
	_, err := stream.Read(b)
	if err != nil {
		log.Printf("Failed to read from connection")
	}

	f.tcpBuf <- b
}

func (f *fakeNode) Start(ctx context.Context) error {
	err := f.tcp.Start(ctx)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (f *fakeNode) Stop(ctx context.Context) error {
	err := f.tcp.Stop(ctx)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func newFakeNode(f Factory) *fakeNode {
	n := &fakeNode{}
	n.tcp, _ = f.CreateStreamTransport(n)

	n.tcpBuf = make(chan []byte, 1)
	return n
}

func (s *suiteTest) TestStreamTransport() {
	ctx := context.Background()
	n1 := newFakeNode(s.factory1)
	n2 := newFakeNode(s.factory2)
	s.NotNil(n2)

	s.NoError(n1.Start(ctx))
	s.NoError(n2.Start(ctx))

	_, err := n2.tcp.Dial(ctx, "127.0.0.1:555555")
	s.Error(err)

	_, err = n2.tcp.Dial(ctx, "invalid address")
	s.Error(err)

	conn, err := n1.tcp.Dial(ctx, n2.tcp.Address())
	s.Require().NoError(err)

	count, err := conn.Write([]byte{1, 2, 3})
	s.Equal(3, count)
	s.NoError(err)
	s.NoError(conn.Close())

	s.Equal([]byte{1, 2, 3}, <-n2.tcpBuf)

	s.NoError(n1.Stop(ctx))
	s.NoError(n2.Stop(ctx))
}

func TestTransport(t *testing.T) {
	instestlogger.SetTestOutput(t)

	cfg1 := configuration.Transport{Protocol: "TCP", Address: "127.0.0.1:0"}
	cfg2 := configuration.Transport{Protocol: "TCP", Address: "127.0.0.1:0"}

	f1 := NewFactory(cfg1)
	f2 := NewFactory(cfg2)
	suite.Run(t, &suiteTest{factory1: f1, factory2: f2})
}
