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

	// GetAccessor get accessor to the internal snapshot for the current pulse. Panics on unknown pulse.
	GetAccessor(pulse.Number) NodeSnapshot
	// GetLatestAccessor returns the latest set of nodes. Will return nil when nothing was added yet.
	GetLatestAccessor() NodeSnapshot
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeKeeper -s _mock.go -g

type NodeKeeper interface {
	NodeNetwork

	AddExpectedBeat(Beat) error
	AddCommittedBeat(Beat) error
}

