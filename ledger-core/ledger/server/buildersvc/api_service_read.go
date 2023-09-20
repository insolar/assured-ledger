package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
)

type ReadService interface {
	DropReadDirty(jet.DropID, func(reader bundle.DirtyReader) error) error
}
