// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/storage.SnapshotStorage -o ../../testutils/network -s _mock.go -g

// SnapshotStorage provides methods for accessing Snapshot.
type SnapshotStorage interface {
	ForPulseNumber(pulse.Number) (*node.Snapshot, error)
	Append(*node.Snapshot) error
}
