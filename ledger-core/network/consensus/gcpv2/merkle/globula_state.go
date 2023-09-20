package merkle

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
)

type GlobulaLeaf struct {
	// ByteSize = 16

	NodeID node.ShortNodeID // ByteSize = 4

	// ByteSize = 4
	NodeRole   member.PrimaryRole // 8
	PowerTotal uint32             // 23

	// ByteSize = 4
	NodePower member.Power // 8
	PowerBase uint32       // 23

	// ByteSize = 4
	RoleIndex uint16        // 10
	RoleTotal uint16        // 10
	NodeTotal uint16        // 10
	OpMode    member.OpMode // 4
}
