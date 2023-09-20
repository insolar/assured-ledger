package catalog

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
)

var _ rmsreg.MarshalerTo = &rms.CatalogEntry{}
var _ bundle.MarshalerTo = &rms.CatalogEntry{}
type Entry = rms.CatalogEntry
