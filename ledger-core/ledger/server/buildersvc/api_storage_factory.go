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
