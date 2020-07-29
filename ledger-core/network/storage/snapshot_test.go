// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNewMemorySnapshotStorage(t *testing.T) {
	ss := NewMemoryStorage()

	ks := platformpolicy.NewKeyProcessor()
	p1, err := ks.GeneratePrivateKey()
	n := node.NewNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, ks.ExtractPublicKey(p1), "127.0.0.1:22")

	pulse := pulsar.PulsePacket{PulseNumber: 15}
	snap := node.NewSnapshot(pulse.PulseNumber, []nodeinfo.NetworkNode{n})

	err = ss.Append(snap)
	assert.NoError(t, err)

	snapshot2, err := ss.ForPulseNumber(pulse.PulseNumber)
	assert.NoError(t, err)

	assert.True(t, snap.Equal(snapshot2))
}
