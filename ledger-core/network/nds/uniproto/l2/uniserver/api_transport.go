package uniserver

import (
	"crypto/tls"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type AllTransportProvider interface {
	CreateSessionlessProvider(binding nwapi.Address, preference nwapi.Preference, maxUDPSize uint16) l1.SessionlessTransportProvider
	CreateSessionfulProvider(binding nwapi.Address, preference nwapi.Preference, tlsCfg *tls.Config) l1.SessionfulTransportProvider
}
