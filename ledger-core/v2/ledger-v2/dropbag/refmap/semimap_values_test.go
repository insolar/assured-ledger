// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package refmap

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestKeys(t *testing.T) {
	m := NewRefLocatorMap()

	const keyCount = int(1e2)
	count := 0
	for i := keyCount; i > 0; i-- {
		refBase := makeLocal(i)
		for j := keyCount; j > 0; j-- {
			refLocal := makeLocal(j)
			ref := reference.NewNoCopy(&refLocal, &refBase)
			refCopy := reference.New(refLocal, refBase)
			require.True(t, reference.Equal(ref, refCopy))
			require.False(t, ref == refCopy, i)

			interned := m.Put(ref, ValueLocator(i))
			// if interned != nil && refCopy != nil {
			count++
			// }
			require.True(t, m.Contains(ref), i)
			{
				refLocalAlt := makeLocal(j + 1e7)
				refAlt := reference.New(refLocalAlt, refBase)
				require.False(t, m.Contains(refAlt), i)
				refAlt = reference.New(refBase, refLocalAlt)
				require.False(t, m.Contains(refAlt), i)
			}

			require.True(t, refLocal.Equal(interned.GetLocal()), i)
			require.True(t, refBase.Equal(interned.GetBase()), i)

			{
				ir := m.Intern(ref)
				require.Equal(t, interned, ir)
				require.True(t, interned.GetLocal() == ir.GetLocal(), i)
				require.True(t, interned.GetBase() == ir.GetBase(), i)
			}
			{
				ir := m.Intern(refCopy)
				require.Equal(t, interned, ir)
				require.True(t, interned.GetLocal() == ir.GetLocal(), i)
				require.True(t, interned.GetBase() == ir.GetBase(), i)
			}
		}
	}
	require.Equal(t, count, m.Len())
	require.Equal(t, keyCount, m.keys.InternedKeyCount()-1 /* zero value is always interned */)
}

func makeLocal(i int) reference.Local {
	h := longbits.Bits224{}
	h[0] = byte(i)
	h[len(h)-1] = byte(i >> 8)
	return reference.NewRecordID(pulse.MinTimePulse+pulse.Number(i), h)
}
