// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

var startTime time.Time

func (n *ServiceNetwork) GetNetworkStatus() network.StatusReply {
	var reply network.StatusReply
	reply.NetworkState = n.Gatewayer.Gateway().GetState()

	na := n.NodeKeeper.GetLatestAccessor()

	var activeNodes, workingNodes []nodeinfo.NetworkNode
	if na != nil {
		reply.PulseNumber = na.GetPulseNumber()
		activeNodes = na.GetActiveNodes()
		workingNodes = na.GetWorkingNodes()
	}

	reply.ActiveListSize = len(activeNodes)
	reply.WorkingListSize = len(workingNodes)

	reply.Nodes = activeNodes
	reply.Origin = n.NodeKeeper.GetOrigin()

	reply.Version = version.Version

	reply.Timestamp = time.Now()
	reply.StartTime = startTime

	return reply
}

func init() {
	startTime = time.Now()
}
