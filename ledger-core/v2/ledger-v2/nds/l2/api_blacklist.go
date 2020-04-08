// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import "github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"

type BlacklistManager interface {
	IsBlacklisted(a l1.Address) bool
	ReportFraud(l1.Address, PeerManager, error) bool
}
