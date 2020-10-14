// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeNetwork -s _mock.go -g

type NodeNetwork interface {
	// GetLocalNodeReference returns a node reference for this/local node. Safe to call at any time. Immutable.
	GetLocalNodeReference() reference.Holder
	// GetLocalNodeRole returns a role for this/local node. Safe to call at any time. Immutable.
	GetLocalNodeRole() member.PrimaryRole

	// GetNodeSnapshot get a snapshot of nodes for the given pulse number. Can return expected snapshot as well. Panics on unknown pulse.
	GetNodeSnapshot(pulse.Number) NodeSnapshot
	// GetAnyLatestNodeSnapshot returns the latest set of nodes. Will return nil when nothing was added yet.
	FindAnyLatestNodeSnapshot() NodeSnapshot
}
