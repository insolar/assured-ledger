// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
)

// NewNodeNetwork create active node component
func NewNodeNetwork(_ configuration.Transport, certificate nodeinfo.Certificate) (network.NodeNetwork, error) {
	nodeKeeper := NewNodeKeeper(certificate.GetNodeRef(), certificate.GetRole())
	return nodeKeeper, nil
}
