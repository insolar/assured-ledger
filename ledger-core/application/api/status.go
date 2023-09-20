package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/insolar/rpc/v2"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// Get returns status info
func (s *NodeService) GetStatus(r *http.Request, args *interface{}, requestBody *rpc.RequestBody, reply *requester.StatusResponse) error {
	traceID := trace.RandID()
	ctx := inslogger.SetLogger(context.Background(), s.runner.logger)
	_, inslog := inslogger.WithTraceField(ctx, traceID)

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
		Reference: reference.Encode(statusReply.LocalRef),
		Role:      statusReply.LocalRole.String(),
	}

	if statusReply.LocalNode != nil {
		reply.Origin.IsWorking = statusReply.LocalNode.IsPowered()
		reply.Origin.ID = uint32(statusReply.LocalNode.GetNodeID())
	}

	reply.NetworkPulseNumber = uint32(statusReply.PulseNumber)

	if p, err := s.runner.PulseAccessor.LatestTimeBeat(); err == nil {
		reply.PulseNumber = uint32(p.PulseNumber)
	} else {
		reply.PulseNumber = pulse.MinTimePulse
	}

	reply.Version = statusReply.Version
	reply.StartTime = statusReply.StartTime
	reply.Timestamp = statusReply.Timestamp

	return nil
}
