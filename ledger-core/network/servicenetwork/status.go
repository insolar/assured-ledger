// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

var startTime time.Time

func (n *ServiceNetwork) GetNetworkStatus() network.StatusReply {
	var reply network.StatusReply
	reply.NetworkState = n.Gatewayer.Gateway().GetState()

	reply.LocalRef = n.NodeKeeper.GetLocalNodeReference()
	reply.LocalRole = n.NodeKeeper.GetLocalNodeRole()

	na := n.NodeKeeper.GetLatestAccessor()

	if na != nil {
		reply.PulseNumber = na.GetPulseNumber()
		reply.WorkingListSize = len(na.GetWorkingNodes())

		activeNodes := na.GetActiveNodes()
		reply.ActiveListSize = len(activeNodes)
		reply.Nodes = activeNodes

		reply.LocalNode = na.GetLocalNode()
	}

	reply.Version = version.Version
	reply.Timestamp = time.Now()
	reply.StartTime = startTime

	return reply
}

func init() {
	startTime = time.Now()
}
