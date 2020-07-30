// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mutable

import (
	"testing"

	"github.com/stretchr/testify/require"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNode_ShortID(t *testing.T) {
	n := NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, nil, "127.0.0.1")
	require.EqualValues(t, node2.GenerateUintShortID(n.GetReference()), n.GetNodeID())
}
