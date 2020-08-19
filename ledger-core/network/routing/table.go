// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package routing

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Table struct {
	NodeKeeper beat.NodeKeeper `inject:""`
}

func (t *Table) isLocalNode(reference.Global) bool {
	return true
}

func (t *Table) resolveRemoteNode(reference.Global) (*host.Host, error) {
	return nil, errors.New("not implemented")
}

// Resolve NodeID -> ShortID, Address. Can initiate network requests.
func (t *Table) Resolve(ref reference.Global) (*host.Host, error) {
	if t.isLocalNode(ref) {
		na := t.NodeKeeper.FindAnyLatestNodeSnapshot()
		if na == nil {
			return nil, errors.E("failed to get latest pulse --==-- ")
		}
		node := na.FindNodeByRef(ref)
		if node == nil {
			return nil, errors.New("no such local node with NodeID: " + ref.String())
		}
		return host.NewHostNS(nodeinfo.NodeAddr(node), ref, node.GetNodeID())
	}
	return t.resolveRemoteNode(ref)
}
