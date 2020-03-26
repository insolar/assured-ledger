// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/server/internal/headless"
	"github.com/insolar/assured-ledger/ledger-core/v2/server/internal/heavy"
	"github.com/insolar/assured-ledger/ledger-core/v2/server/internal/light"
	"github.com/insolar/assured-ledger/ledger-core/v2/server/internal/virtual"
)

type Server interface {
	Serve()
}

func NewLightServer(cfgPath string) Server {
	return light.New(cfgPath)
}

func NewHeavyServer(cfgPath string, gensisCfgPath string) Server {
	return heavy.New(cfgPath, gensisCfgPath)
}

func NewVirtualServer(cfgPath string) Server {
	return virtual.New(cfgPath)
}

func NewHeadlessNetworkNodeServer(cfgPath string) Server {
	return headless.New(cfgPath)
}
