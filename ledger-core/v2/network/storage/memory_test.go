// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func TestMemoryStorage(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()
	startPulse := *pulsestor.GenesisPulse

	ks := platformpolicy.NewKeyProcessor()
	p1, err := ks.GeneratePrivateKey()
	assert.NoError(t, err)
	n := node.NewNode(gen.Reference(), node2.StaticRoleVirtual, ks.ExtractPublicKey(p1), "127.0.0.1:22", "ver2")
	nodes := []node2.NetworkNode{n}

	for i := 0; i < entriesCount+2; i++ {
		p := startPulse
		p.PulseNumber += pulse.Number(i)

		snap := node.NewSnapshot(p.PulseNumber, nodes)
		err = s.Append(p.PulseNumber, snap)
		assert.NoError(t, err)

		err = s.AppendPulse(ctx, p)
		assert.NoError(t, err)

		p1, err := s.GetLatestPulse(ctx)
		assert.NoError(t, err)
		assert.Equal(t, p, p1)

		snap1, err := s.ForPulseNumber(p1.PulseNumber)
		assert.NoError(t, err)
		assert.True(t, snap1.Equal(snap), "snapshots should be equal")
	}

	// first pulse and snapshot should be truncated
	assert.Len(t, s.entries, entriesCount)
	assert.Len(t, s.snapshotEntries, entriesCount)

	p2, err := s.GetPulse(ctx, startPulse.PulseNumber)
	assert.EqualError(t, err, ErrNotFound.Error())
	assert.Equal(t, p2, *pulsestor.GenesisPulse)

	snap2, err := s.ForPulseNumber(startPulse.PulseNumber)
	assert.EqualError(t, err, ErrNotFound.Error())
	assert.Nil(t, snap2)

}
