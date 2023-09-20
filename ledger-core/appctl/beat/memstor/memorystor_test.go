package memstor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func TestMemoryStorage(t *testing.T) {
	s := NewMemoryStorage()
	startPulse := pulse.Number(pulse.MinTimePulse)

	for i := startPulse; i < startPulse+entriesCount+2; i++ {
		snap := NewSnapshot(i, nil)
		assert.NoError(t, s.Append(snap))

		snap1, err := s.ForPulseNumber(i)
		assert.NoError(t, err)
		assert.True(t, snap1 == snap, "snapshots should be equal")
	}

	// first pulse and snapshot should be truncated
	assert.Len(t, s.entries, entriesCount)
	assert.Len(t, s.snapshotEntries, entriesCount)

	snap, err := s.ForPulseNumber(startPulse)
	assert.EqualError(t, err, ErrNotFound.Error())
	assert.Nil(t, snap)

	snap, err = s.ForPulseNumber(startPulse + 1)
	assert.EqualError(t, err, ErrNotFound.Error())
	assert.Nil(t, snap)

	snap, err = s.ForPulseNumber(startPulse + 2)
	assert.Nil(t, err)
	assert.NotNil(t, snap)
}
