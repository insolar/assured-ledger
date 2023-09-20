package profiles

import "github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"

type PopulationRank struct {
	Profile ActiveNode
	Power   member.Power
	OpMode  member.OpMode
}
