// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPulseGenerator(t *testing.T) {
	delta := uint16(13)
	pg := NewPulseGenerator(delta, nil, nil)

	require.Equal(t, delta, pg.GetDelta())

	require.Panics(t, func() { pg.GetLastPulsePacket() })

	{
		prevPulseData := pg.Generate()
		require.Equal(t, prevPulseData, pg.GetLastPulseData())
		_ = pg.Generate()
		require.Equal(t, prevPulseData, pg.GetPrevBeat().Data)
	}

	lastPacket := pg.GetLastPulsePacket()
	packet := lastPacket
	require.Equal(t, packet, lastPacket)

	{
		_ = pg.Generate()
		lastPacket = pg.GetLastPulsePacket()
		require.NotEqual(t, lastPacket, packet)
		require.Equal(t, delta, uint16(lastPacket.PulseNumber-packet.PulseNumber))

		lp := pg.GetLastPulsePacket()
		require.Equal(t, lp, lastPacket)
	}

	{
		back0Packet := pg.CountBackPacket(0)
		require.Equal(t, lastPacket, back0Packet)
		require.Equal(t, pg.CountBackPacket(0), back0Packet)

		back1Packet := pg.CountBackPacket(1)
		require.Equal(t, back1Packet, packet)

		require.Equal(t, back1Packet, pg.CountBackPacket(1))

		require.Panics(t, func() { pg.CountBackPacket(3) })

		require.Panics(t, func() { pg.CountBackPacket(200) })
	}
}
