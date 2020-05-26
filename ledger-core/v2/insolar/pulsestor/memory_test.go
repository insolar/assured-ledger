// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
)

func TestNodeStorage_ForPulseNumber(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	pn := gen.PulseNumber()
	pulse := Pulse{PulseNumber: pn}
	storage := NewStorageMem()
	storage.storage[pn] = &memNode{pulse: pulse}

	t.Run("returns error when no Pulse", func(t *testing.T) {
		t.Skip("fixme")
		res, err := storage.ForPulseNumber(ctx, gen.PulseNumber())
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, Pulse{}, res)
	})

	t.Run("returns correct Pulse", func(t *testing.T) {
		res, err := storage.ForPulseNumber(ctx, pn)
		assert.NoError(t, err)
		assert.Equal(t, pulse, res)
	})
}

func TestNodeStorage_Latest(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)

	t.Run("returns error when no Pulse", func(t *testing.T) {
		storage := NewStorageMem()
		res, err := storage.Latest(ctx)
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, Pulse{}, res)
	})

	t.Run("returns correct Pulse", func(t *testing.T) {
		storage := NewStorageMem()
		pulse := Pulse{PulseNumber: gen.PulseNumber()}
		storage.tail = &memNode{pulse: pulse}
		res, err := storage.Latest(ctx)
		assert.NoError(t, err)
		assert.Equal(t, pulse, res)
	})
}

func TestNodeStorage_Append(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	pn := gen.PulseNumber()
	puls := Pulse{PulseNumber: pn}

	t.Run("appends to an empty storage", func(t *testing.T) {
		storage := NewStorageMem()

		err := storage.Append(ctx, puls)
		require.NoError(t, err)
		require.NotNil(t, storage.tail)
		require.NotNil(t, storage.storage[puls.PulseNumber])
		assert.Equal(t, storage.storage[puls.PulseNumber], storage.tail)
		assert.Equal(t, storage.head, storage.tail)
		assert.Equal(t, memNode{pulse: puls}, *storage.tail)
	})

	t.Run("returns error if Pulse number is equal or less", func(t *testing.T) {
		storage := NewStorageMem()
		head := &memNode{pulse: puls}
		storage.storage[pn] = head
		storage.tail = head
		storage.head = head

		{
			err := storage.Append(ctx, Pulse{PulseNumber: pn})
			assert.Equal(t, ErrBadPulse, err)
		}
		{
			err := storage.Append(ctx, Pulse{PulseNumber: pn - 1})
			assert.Equal(t, ErrBadPulse, err)
		}
	})

	t.Run("appends to a filled storage", func(t *testing.T) {
		storage := NewStorageMem()
		head := &memNode{pulse: puls}
		storage.storage[pn] = head
		storage.tail = head
		storage.head = head
		pulse := puls
		pulse.PulseNumber += 1

		err := storage.Append(ctx, pulse)
		require.NoError(t, err)
		require.NotNil(t, storage.tail)
		require.NotNil(t, storage.storage[pulse.PulseNumber])
		assert.Equal(t, storage.storage[pulse.PulseNumber], storage.tail)
		assert.NotEqual(t, storage.head, storage.tail)
		assert.Equal(t, memNode{pulse: pulse, prev: head}, *storage.tail)
	})

	t.Run("appends 5 pulses", func(t *testing.T) {
		storage := NewStorageMem()

		err := storage.Append(ctx, Pulse{PulseNumber: 1})
		require.NoError(t, err)
		err = storage.Append(ctx, Pulse{PulseNumber: 2})
		require.NoError(t, err)
		err = storage.Append(ctx, Pulse{PulseNumber: 3})
		require.NoError(t, err)
		err = storage.Append(ctx, Pulse{PulseNumber: 4})
		require.NoError(t, err)
		err = storage.Append(ctx, Pulse{PulseNumber: 5})
		require.NoError(t, err)

		require.Equal(t, 5, len(storage.storage))
		_, ok := storage.storage[1]
		require.Equal(t, true, ok)
		_, ok = storage.storage[2]
		require.Equal(t, true, ok)
		_, ok = storage.storage[3]
		require.Equal(t, true, ok)
		_, ok = storage.storage[4]
		require.Equal(t, true, ok)
		_, ok = storage.storage[5]
		require.Equal(t, true, ok)

		require.Equal(t, pulse.Number(1), storage.head.pulse.PulseNumber)
		require.Equal(t, pulse.Number(5), storage.tail.pulse.PulseNumber)

		pn := 1
		cur := storage.head

		for cur != nil {
			require.Equal(t, pulse.Number(pn), cur.pulse.PulseNumber)
			cur = cur.next
			pn++
		}

	})
}

func TestMemoryStorage_Shift(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	pn := gen.PulseNumber()
	puls := Pulse{PulseNumber: pn}

	t.Run("returns error if empty", func(t *testing.T) {
		storage := NewStorageMem()
		err := storage.Shift(ctx, pn)
		assert.Error(t, err)
	})

	t.Run("shifts if one in storage", func(t *testing.T) {
		storage := NewStorageMem()
		head := &memNode{pulse: puls}
		storage.storage[pn] = head
		storage.tail = head
		storage.head = head

		err := storage.Shift(ctx, pn)
		assert.NoError(t, err)
		assert.Nil(t, storage.tail)
		assert.Nil(t, storage.head)
		assert.Empty(t, storage.storage)
	})

	t.Run("shifts if two in storage", func(t *testing.T) {
		storage := NewStorageMem()
		tailPulse := puls
		headPulse := puls
		headPulse.PulseNumber += 1
		head := &memNode{pulse: headPulse}
		tail := &memNode{pulse: tailPulse}
		head.prev = tail
		tail.next = head
		storage.storage[headPulse.PulseNumber] = head
		storage.storage[tailPulse.PulseNumber] = tail
		storage.tail = head
		storage.head = tail

		err := storage.Shift(ctx, pn)
		assert.NoError(t, err)
		assert.Equal(t, storage.tail, storage.head)
		assert.Equal(t, head, storage.storage[head.pulse.PulseNumber])
		assert.Equal(t, memNode{pulse: headPulse}, *head)
	})

	t.Run("shifts middle, when 5 pulses", func(t *testing.T) {
		storage := NewStorageMem()

		_ = storage.Append(ctx, Pulse{PulseNumber: 101})
		_ = storage.Append(ctx, Pulse{PulseNumber: 102})
		_ = storage.Append(ctx, Pulse{PulseNumber: 103})
		_ = storage.Append(ctx, Pulse{PulseNumber: 104})
		_ = storage.Append(ctx, Pulse{PulseNumber: 105})

		err := storage.Shift(ctx, 103)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(storage.storage))
		_, ok := storage.storage[104]
		require.Equal(t, true, ok)
		_, ok = storage.storage[105]
		require.Equal(t, true, ok)

		assert.Equal(t, pulse.Number(104), storage.head.pulse.PulseNumber)
		assert.Equal(t, pulse.Number(105), storage.tail.pulse.PulseNumber)
	})
}

func TestMemoryStorage_ForwardsBackwards(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	storage := NewStorageMem()
	tailPulse := Pulse{PulseNumber: gen.PulseNumber()}
	headPulse := Pulse{PulseNumber: tailPulse.PulseNumber + 1}
	head := &memNode{pulse: headPulse}
	tail := &memNode{pulse: tailPulse}
	head.prev = tail
	tail.next = head
	storage.storage[headPulse.PulseNumber] = head
	storage.storage[tailPulse.PulseNumber] = tail
	storage.tail = head
	storage.head = tail

	t.Run("forwards returns itself if zero steps", func(t *testing.T) {
		pulse, err := storage.Forwards(ctx, tailPulse.PulseNumber, 0)
		assert.NoError(t, err)
		assert.Equal(t, pulse, tailPulse)
	})
	t.Run("forwards returns Next if one step", func(t *testing.T) {
		pulse, err := storage.Forwards(ctx, tailPulse.PulseNumber, 1)
		assert.NoError(t, err)
		assert.Equal(t, pulse, headPulse)
	})
	t.Run("forwards returns error if forward overflow", func(t *testing.T) {
		pulse, err := storage.Forwards(ctx, tailPulse.PulseNumber, 2)
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, Pulse{}, pulse)
	})
	t.Run("forwards returns error if backward overflow", func(t *testing.T) {
		pulse, err := storage.Forwards(ctx, tailPulse.PulseNumber-1, 1)
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, Pulse{}, pulse)
	})

	t.Run("backwards returns itself if zero steps", func(t *testing.T) {
		pulse, err := storage.Backwards(ctx, headPulse.PulseNumber, 0)
		assert.NoError(t, err)
		assert.Equal(t, pulse, headPulse)
	})
	t.Run("backwards returns Next if one step", func(t *testing.T) {
		pulse, err := storage.Backwards(ctx, headPulse.PulseNumber, 1)
		assert.NoError(t, err)
		assert.Equal(t, pulse, tailPulse)
	})
	t.Run("backwards returns error if backward overflow", func(t *testing.T) {
		pulse, err := storage.Backwards(ctx, headPulse.PulseNumber, 2)
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, Pulse{}, pulse)
	})
	t.Run("backwards returns error if forward overflow", func(t *testing.T) {
		pulse, err := storage.Backwards(ctx, headPulse.PulseNumber-1, 1)
		assert.Equal(t, ErrNotFound, err)
		assert.Equal(t, Pulse{}, pulse)
	})
}
