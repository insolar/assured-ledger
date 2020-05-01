// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/storage.SnapshotStorage -o ../../testutils/network -s _mock.go -g

// SnapshotStorage provides methods for accessing Snapshot.
type SnapshotStorage interface {
	ForPulseNumber(insolar.PulseNumber) (*node.Snapshot, error)
	Append(pulse insolar.PulseNumber, snapshot *node.Snapshot) error
}
