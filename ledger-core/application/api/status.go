// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/insolar/rpc/v2"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
)

// Get returns status info
func (s *NodeService) GetStatus(r *http.Request, args *interface{}, requestBody *rpc.RequestBody, reply *requester.StatusResponse) error {
	traceID := trace.RandID()
	ctx, inslog := inslogger.WithTraceField(context.Background(), traceID)

	inslog.Infof("[ NodeService.GetStatus ] Incoming request: %s", r.RequestURI)
	if !s.runner.cfg.IsAdmin {
		return errors.New("method not allowed")
	}
	statusReply := s.runner.NetworkStatus.GetNetworkStatus()
	inslog.Debugf("[ NodeService.GetStatus ] Current status: %v", statusReply.NetworkState)

	reply.NetworkState = statusReply.NetworkState.String()
	reply.ActiveListSize = statusReply.ActiveListSize
	reply.WorkingListSize = statusReply.WorkingListSize

	nodes := make([]requester.Node, reply.ActiveListSize)
	for i, node := range statusReply.Nodes {
		nodes[i] = requester.Node{
			Reference: nodeinfo.NodeRef(node).String(),
			Role:      nodeinfo.NodeRole(node).String(),
			IsWorking: node.IsPowered(),
			ID:        uint32(node.GetNodeID()),
		}
	}
	reply.Nodes = nodes

	reply.Origin = requester.Node{
		Reference: nodeinfo.NodeRef(statusReply.Origin).String(),
		Role:      nodeinfo.NodeRole(statusReply.Origin).String(),
		IsWorking: statusReply.Origin.IsPowered(),
		ID:        uint32(statusReply.Origin.GetNodeID()),
	}

	reply.NetworkPulseNumber = uint32(statusReply.PulseNumber)

	p, err := s.runner.PulseAccessor.Latest(ctx)
	if err != nil {
		p = pulsestor.GenesisPulse
	}
	reply.PulseNumber = uint32(p.PulseNumber)

	reply.Version = statusReply.Version
	reply.StartTime = statusReply.StartTime
	reply.Timestamp = statusReply.Timestamp

	return nil
}
