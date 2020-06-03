// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type BlacklistManager interface {
	IsBlacklisted(a nwapi.Address) bool
	ReportFraud(nwapi.Address, *PeerManager, error) bool
}
