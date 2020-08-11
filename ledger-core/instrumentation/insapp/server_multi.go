// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type NetworkSupport interface {
	network.NodeNetwork
	nodeinfo.CertificateGetter

	GetSendMessageHandler() message.NoPublishHandlerFunc
}

type NetworkInitFunc = func(configuration.Configuration, *component.Manager) (NetworkSupport, network.Status, error)
type MultiNodeConfigFunc = func(cfgPath string, baseCfg configuration.Configuration) ([]configuration.Configuration, NetworkInitFunc)

func NewMulti(cfgPath string, appFn AppFactoryFunc, multiFn MultiNodeConfigFunc) *Server {
	switch {
	case appFn == nil:
		panic(throw.IllegalValue())
	case multiFn == nil:
		panic(throw.IllegalValue())
	}

	return &Server{
		cfgPath: cfgPath,
		appFn:   appFn,
		multiFn: multiFn,
	}
}

