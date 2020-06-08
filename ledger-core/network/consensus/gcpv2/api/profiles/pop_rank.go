// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package profiles

import "github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"

type PopulationRank struct {
	Profile ActiveNode
	Power   member.Power
	OpMode  member.OpMode
}
