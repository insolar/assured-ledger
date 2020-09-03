// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
)

type Server interface {
	Serve()
}

func NewNode(cfg configuration.Configuration) Server {
	return insapp.New(cfg)
}

func NewMultiServer(cfg configuration.BaseCloudConfig, multiFn insapp.MultiNodeConfigFunc) Server {
	return insapp.NewMulti(cfg, multiFn)
}

func NewHeadlessNetworkNodeServer(cfg configuration.Configuration) Server {
	return insapp.NewHeadless(cfg)
}
