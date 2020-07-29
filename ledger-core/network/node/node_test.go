// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"

	"github.com/stretchr/testify/assert"
)

func TestNode_ShortID(t *testing.T) {
	n := NewNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, nil, "127.0.0.1")
	assert.EqualValues(t, GenerateUintShortID(n.GetReference()), n.GetNodeID())
	n.(MutableNode).SetShortID(11)
	assert.EqualValues(t, 11, n.GetNodeID())
}
