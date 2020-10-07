// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type StorageFactory interface {
	CreateSnapshotWriter(pn pulse.Number, maxSection ledger.SectionID) bundle.SnapshotWriter
	DepositReadOnlyWriter(bundle.SnapshotWriter) error
}
