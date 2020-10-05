// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
)

func TestTTLMap(t *testing.T) {
	ttlMap := ttlMap{}

	sh0 := msgShipment{
		id:      ShipmentID(1),
		expires: 0,
	}
	sh1 := msgShipment{
		id:      ShipmentID(2),
		expires: 1,
	}
	sh2 := msgShipment{
		id:      ShipmentID(3),
		expires: 2,
	}
	shN := msgShipment{
		id:      ShipmentID(4),
		expires: 42,
	}

	ttlMap.put(&sh0, 0)
	ttlMap.put(&sh1, 0)
	ttlMap.put(&shN, 0)

	require.Equal(t, 1, len(ttlMap.ttl0))
	require.Equal(t, 1, len(ttlMap.ttl1))
	require.Equal(t, 1, len(ttlMap.ttlN))

	require.Equal(t, &sh0, ttlMap.get(ShipmentID(1)))
	require.Equal(t, &sh1, ttlMap.get(ShipmentID(2)))
	require.Equal(t, &shN, ttlMap.get(ShipmentID(4)))

	ttlMap.deleteAll([]retries.RetryID{})

	require.Equal(t, 1, len(ttlMap.ttl0))
	require.Equal(t, 1, len(ttlMap.ttl1))
	require.Equal(t, 1, len(ttlMap.ttlN))

	require.Equal(t, &sh0, ttlMap.get(ShipmentID(1)))
	require.Equal(t, &sh1, ttlMap.get(ShipmentID(2)))
	require.Equal(t, &shN, ttlMap.get(ShipmentID(4)))

	ttlMap.deleteAll([]retries.RetryID{
		retries.RetryID(1),
		retries.RetryID(2),
		retries.RetryID(4),
	})

	require.Nil(t, ttlMap.get(ShipmentID(1)))
	require.Nil(t, ttlMap.get(ShipmentID(2)))
	require.Nil(t, ttlMap.get(ShipmentID(4)))

	ttlMap.put(&sh0, 0)
	ttlMap.put(&sh1, 0)
	ttlMap.put(&sh2, 0)

	ttlMap.nextTTLCycle(1)
	ttlMap.sinkAfterCycle()

	require.Equal(t, 1, len(ttlMap.ttl0))
	require.Equal(t, 1, len(ttlMap.ttl1))
	require.Equal(t, 0, len(ttlMap.ttlN))

	ttlMap.put(&shN, 0)

	for i := 1; i < 41; i++ {
		ttlMap.nextTTLCycle(uint32(i))
		ttlMap.sinkAfterCycle()
		require.Equal(t, 1, len(ttlMap.ttlN))
	}

	ttlMap.nextTTLCycle(41)
	ttlMap.sinkAfterCycle()

	require.Equal(t, 0, len(ttlMap.ttl0))
	require.Equal(t, 1, len(ttlMap.ttl1))
	require.Equal(t, 0, len(ttlMap.ttlN))

	ttlMap.nextTTLCycle(42)
	ttlMap.sinkAfterCycle()

	require.Equal(t, 1, len(ttlMap.ttl0))
	require.Equal(t, 0, len(ttlMap.ttl1))
	require.Equal(t, 0, len(ttlMap.ttlN))

	ttlMap.nextTTLCycle(43)
	ttlMap.sinkAfterCycle()

	require.Equal(t, 0, len(ttlMap.ttl0))
	require.Equal(t, 0, len(ttlMap.ttl1))
	require.Equal(t, 0, len(ttlMap.ttlN))
}

//nolint
func (p *ttlMap) sinkAfterCycle() {
	p.mutex.Lock() // make sure that processing is done
	p.mutex.Unlock()
}
