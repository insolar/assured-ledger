// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmntestapp

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor/instestconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

func NewTestServer(t logcommon.TestingLogger) *TestServer {
	return NewTestServerWithErrorFilter(t, func(string) bool { return false })
}

func NewTestServerWithErrorFilter(t logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) *TestServer {
	s := instestconveyor.NewTestServerTemplate(t, filterFn)
	s.InitTemplate(
		func(c configuration.Configuration, comps insapp.AppComponents) *insconveyor.AppCompartment {
			return lmnapp.NewAppCompartment(c.Ledger, comps)
		}, nil)
	return &TestServer{s}
}

type TestServer struct {
	*instestconveyor.ServerTemplate
}

func (p *TestServer) SetConfig(cfg configuration.Ledger) {
	p.ServerTemplate.SetConfig(configuration.Configuration{
		Ledger: cfg,
	})
}
