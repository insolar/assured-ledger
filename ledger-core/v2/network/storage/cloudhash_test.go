// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

func TestInMemoryCloudHashStorage(t *testing.T) {
	cs := NewMemoryCloudHashStorage()

	pulse := insolar.Pulse{PulseNumber: 15}
	cloudHash := []byte{1, 2, 3, 4, 5}

	err := cs.Append(pulse.PulseNumber, cloudHash)
	assert.NoError(t, err)

	cloudHash2, err := cs.ForPulseNumber(pulse.PulseNumber)
	assert.NoError(t, err)

	assert.Equal(t, cloudHash, cloudHash2)
}
