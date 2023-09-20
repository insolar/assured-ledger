package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type BlacklistManager interface {
	IsBlacklisted(a nwapi.Address) bool
	ReportFraud(nwapi.Address, *PeerManager, error) bool
}
