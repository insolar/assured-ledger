// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNodeStorage_ForPulseNumber(t *testing.T) {
	instestlogger.SetTestOutput(t)

	pn := gen.PulseNumber()
	pc := beat.Beat{}
	pc.PulseNumber = pn

	storage := NewStorageMem()
	storage.storage[pn] = &memNode{pulse: pc}

	t.Run("returns error when no Pulse", func(t *testing.T) {
		res, err := storage.TimeBeat(pc.PulseNumber + 1)
		assert.Equal(t, ErrNotFound.Error(), err.Error())
		assert.Equal(t, beat.Beat{}, res)
	})

	t.Run("returns correct Pulse", func(t *testing.T) {
		res, err := storage.TimeBeat(pn)
		assert.NoError(t, err)
		assert.Equal(t, pc, res)
	})
}

func TestNodeStorage_Latest(t *testing.T) {
	instestlogger.SetTestOutput(t)

	t.Run("returns error when no Pulse", func(t *testing.T) {
		storage := NewStorageMem()
		res, err := storage.LatestTimeBeat()
		assert.Equal(t, ErrNotFound.Error(), err.Error())
		assert.Equal(t, beat.Beat{}, res)
	})

	t.Run("returns correct Pulse", func(t *testing.T) {
		storage := NewStorageMem()
		pn := gen.PulseNumber()
		pc := beat.Beat{}
		pc.PulseNumber = pn
		storage.tail = &memNode{pulse: pc}
		res, err := storage.LatestTimeBeat()
		assert.NoError(t, err)
		assert.Equal(t, pc, res)
	})
}

func TestNodeStorage_Append(t *testing.T) {
	instestlogger.SetTestOutput(t)

	pn := gen.PulseNumber()
	pc := beat.Beat{}
	pc.PulseNumber = pn
	pc.PulseEpoch = pn.AsEpoch()

	t.Run("appends to an empty storage", func(t *testing.T) {
		storage := NewStorageMem()

		err := storage.AddCommittedBeat(pc)
		require.NoError(t, err)
		require.NotNil(t, storage.tail)
		require.NotNil(t, storage.storage[pc.PulseNumber])
		assert.Equal(t, storage.storage[pc.PulseNumber], storage.tail)
		assert.Equal(t, storage.head, storage.tail)
		assert.Equal(t, memNode{pulse: pc}, *storage.tail)
	})

	t.Run("returns error if Pulse number is equal or less", func(t *testing.T) {
		storage := NewStorageMem()
		head := &memNode{pulse: pc}
		storage.storage[pn] = head
		storage.tail = head
		storage.head = head

		{
			err := storage.AddCommittedBeat(pc)
			assert.Equal(t, ErrBadPulse.Error(), err.Error())
		}
		{
			pc1 := pc
			pc1.PulseNumber--

			err := storage.AddCommittedBeat(pc1)
			assert.Equal(t, ErrBadPulse.Error(), err.Error())
		}
	})

	t.Run("appends to a filled storage", func(t *testing.T) {
		storage := NewStorageMem()
		head := &memNode{pulse: pc}
		storage.storage[pn] = head
		storage.tail = head
		storage.head = head

		pc1 := pc
		pc1.PulseNumber++

		err := storage.AddCommittedBeat(pc1)
		require.NoError(t, err)
		require.NotNil(t, storage.tail)
		require.NotNil(t, storage.storage[pc1.PulseNumber])
		assert.Equal(t, storage.storage[pc1.PulseNumber], storage.tail)
		assert.NotEqual(t, storage.head, storage.tail)
		assert.Equal(t, memNode{pulse: pc1, prev: head}, *storage.tail)
	})

	t.Run("appends 5 pulses", func(t *testing.T) {
		storage := NewStorageMem()

		pc := beat.Beat{}
		pc.PulseNumber = pulse.MinTimePulse
		pc.PulseEpoch = pc.PulseNumber.AsEpoch()

		for i := 5; i > 0; i-- {
			err := storage.AddCommittedBeat(pc)
			require.NoError(t, err)
			pc.PulseNumber++
			pc.PulseEpoch++
		}

		require.Equal(t, 5, len(storage.storage))
		for i := 5; i > 0; i-- {
			pc.PulseNumber--
			_, ok := storage.storage[pc.PulseNumber]
			require.True(t, ok)
		}

		require.Equal(t, pulse.Number(pulse.MinTimePulse), storage.head.pulse.PulseNumber)
		require.Equal(t, pulse.Number(pulse.MinTimePulse + 4), storage.tail.pulse.PulseNumber)

		pn := pulse.MinTimePulse
		cur := storage.head

		for cur != nil {
			require.Equal(t, pulse.Number(pn), cur.pulse.PulseNumber)
			cur = cur.next
			pn++
		}
	})
}

func TestMemoryStorage_Shift(t *testing.T) {
	instestlogger.SetTestOutput(t)

	pn := gen.PulseNumber()
	pc := beat.Beat{}
	pc.PulseNumber = pn

	t.Run("returns error if empty", func(t *testing.T) {
		storage := NewStorageMem()
		err := storage.Trim(pn)
		assert.Error(t, err)
	})

	t.Run("shifts if one in storage", func(t *testing.T) {
		storage := NewStorageMem()
		head := &memNode{pulse: pc}
		storage.storage[pn] = head
		storage.tail = head
		storage.head = head

		err := storage.Trim(pn)
		assert.NoError(t, err)
		assert.Nil(t, storage.tail)
		assert.Nil(t, storage.head)
		assert.Empty(t, storage.storage)
	})

	t.Run("shifts if two in storage", func(t *testing.T) {
		storage := NewStorageMem()
		tailPulse := pc
		headPulse := pc
		headPulse.PulseNumber += 1
		head := &memNode{pulse: headPulse}
		tail := &memNode{pulse: tailPulse}
		head.prev = tail
		tail.next = head
		storage.storage[headPulse.PulseNumber] = head
		storage.storage[tailPulse.PulseNumber] = tail
		storage.tail = head
		storage.head = tail

		err := storage.Trim(pn)
		assert.NoError(t, err)
		assert.Equal(t, storage.tail, storage.head)
		assert.Equal(t, head, storage.storage[head.pulse.PulseNumber])
		assert.Equal(t, memNode{pulse: headPulse}, *head)
	})

	t.Run("shifts middle, when 5 pulses", func(t *testing.T) {
		storage := NewStorageMem()

		pc := beat.Beat{}
		pc.PulseNumber = pulse.MinTimePulse
		pc.PulseEpoch = pc.PulseNumber.AsEpoch()

		for i := 5; i > 0; i-- {
			err := storage.AddCommittedBeat(pc)
			require.NoError(t, err)
			pc.PulseNumber++
			pc.PulseEpoch++
		}

		err := storage.Trim(pulse.MinTimePulse + 2)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(storage.storage))
		_, ok := storage.storage[pulse.MinTimePulse + 3]
		require.Equal(t, true, ok)
		_, ok = storage.storage[pulse.MinTimePulse + 4]
		require.Equal(t, true, ok)

		assert.Equal(t, pulse.Number(pulse.MinTimePulse + 3), storage.head.pulse.PulseNumber)
		assert.Equal(t, pulse.Number(pulse.MinTimePulse + 4), storage.tail.pulse.PulseNumber)
	})
}
