// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/headless"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/virtual"
)

type Server interface {
	Serve()
}

func NewLightMaterialServer(cfgPath string) Server {
	return insapp.New(cfgPath, lmnapp.AppFactory)
}

func NewVirtualServer(cfgPath string) Server {
	return insapp.New(cfgPath, virtual.AppFactory)
}

func NewMultiServer(cfgPath string, multiFn insapp.MultiNodeConfigFunc) Server {
	return insapp.NewMulti(cfgPath, virtual.AppFactory, multiFn)
}

func NewHeadlessNetworkNodeServer(cfgPath string) Server {
	return insapp.New(cfgPath, nil, &headless.AppComponent{})
}
