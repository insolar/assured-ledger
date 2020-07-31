// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func TestMemoryStorage(t *testing.T) {
	s := NewMemoryStorage()
	startPulse := pulsestor.GenesisPulse

	for i := 0; i < entriesCount+2; i++ {
		p := startPulse
		p.PulseNumber += pulse.Number(i)

		snap := NewSnapshot(p.PulseNumber, nil)
		assert.NoError(t, s.Append(snap))

		snap1, err := s.ForPulseNumber(p.PulseNumber)
		assert.NoError(t, err)
		assert.True(t, snap1 == snap, "snapshots should be equal")
	}

	// first pulse and snapshot should be truncated
	assert.Len(t, s.entries, entriesCount)
	assert.Len(t, s.snapshotEntries, entriesCount)

	snap, err := s.ForPulseNumber(startPulse.PulseNumber)
	assert.EqualError(t, err, ErrNotFound.Error())
	assert.Nil(t, snap)

	snap, err = s.ForPulseNumber(startPulse.PulseNumber + 1)
	assert.EqualError(t, err, ErrNotFound.Error())
	assert.Nil(t, snap)

	snap, err = s.ForPulseNumber(startPulse.PulseNumber + 2)
	assert.Nil(t, err)
	assert.NotNil(t, snap)
}
