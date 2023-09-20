package routing

import (
	"strconv"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newNode(ref reference.Global, id int) *mutable.Node {
	address := "127.0.0.1:" + strconv.Itoa(id)
	result := mutable.NewTestNode(ref, member.PrimaryRoleUnknown, address)
	result.SetShortID(node.ShortNodeID(id))
	return result
}

func TestTable_Resolve(t *testing.T) {
	table := Table{}

	refs := gen.UniqueGlobalRefs(2)

	vf := cryptkit.NewSignatureVerifierFactoryMock(t)
	vf.CreateSignatureVerifierWithPKSMock.Return(nil)
	pop := censusimpl.NewJoinerPopulation(newNode(refs[0], 123).GetStatic(), vf)
	na := memstor.NewAccessor(memstor.NewSnapshot(pulse.MinTimePulse, &pop))

	nodeKeeperMock := beat.NewNodeKeeperMock(t)
	nodeKeeperMock.GetNodeSnapshotMock.Return(na)
	nodeKeeperMock.FindAnyLatestNodeSnapshotMock.Return(na)

	table.NodeKeeper = nodeKeeperMock

	h, err := table.Resolve(refs[0])
	require.NoError(t, err)
	assert.EqualValues(t, 123, h.ShortID)
	assert.Equal(t, "127.0.0.1:123", h.Address.String())

	_, err = table.Resolve(refs[1])
	assert.Error(t, err)
}
