// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmntestapp

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor/instestconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestServer(t logcommon.TestingLogger) *TestServer {
	return NewTestServerWithErrorFilter(t, func(string) bool { return true })
}

func NewTestServerWithErrorFilter(t logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) *TestServer {
	s := instestconveyor.NewTestServerTemplate(t, filterFn)
	s.InitTemplate(
		func(c configuration.Configuration, comps insapp.AppComponents) *insconveyor.AppCompartment {
			comps.LocalNodeRole = member.PrimaryRoleLightMaterial
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

func (p *TestServer) RunGenesis() {
	var treeSvc treesvc.Service
	p.Injector().MustInject(&treeSvc)

	_, _, ok := treeSvc.GetTrees(pulse.Unknown)
	if !ok {
		// not a genesis
		panic(throw.IllegalState())
	}
	p.IncrementPulse() // trigger genesis

	for !treeSvc.IsGenesisFinished() {
		time.Sleep(time.Millisecond)
	}
}
