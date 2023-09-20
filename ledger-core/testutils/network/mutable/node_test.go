package mutable

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNode_ShortID(t *testing.T) {
	n := NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, "")
	require.Equal(t, node.GenerateShortID(n.GetReference()), n.GetNodeID())
}
