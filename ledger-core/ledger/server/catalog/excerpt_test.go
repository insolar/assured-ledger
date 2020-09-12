// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
)

func TestExcerptTagSize(t *testing.T) {
	require.Equal(t, protokit.WireVarint.Tag(MaxExcerptFieldID).TagSize(), excerptTagSize)
}

func TestExcerptReadFromLazy(t *testing.T) {
	msg := rms.LRegisterRequest{}
	rec := rms.ROutboundRequest{
		PrevRef:    rms.NewReference(gen.UniqueGlobalRef()),
		RootRef:    rms.NewReference(gen.UniqueGlobalRef()),
		Str:        "abc",
	}
	msg.Set(&rec)

	b, err := msg.Marshal()
	require.NoError(t, err)

	_, msgU, err := rmsreg.Unmarshal(b)
	require.NoError(t, err)
	require.IsType(t, &rms.LRegisterRequest{}, msgU)

	msg2 := msgU.(*rms.LRegisterRequest)
	lazyRec := msg2.TryGetLazy()
	require.False(t, lazyRec.IsZero())
	require.False(t, lazyRec.IsEmpty())

	excerpt, err := ReadExcerptFromLazy(lazyRec)
	require.NoError(t, err)

	require.Equal(t, rec.GetDefaultPolymorphID(), uint64(excerpt.RecordType))
	require.Equal(t, rec.PrevRef, excerpt.PrevRef)
	require.Equal(t, rec.RootRef, excerpt.RootRef)
	require.True(t, excerpt.ReasonRef.IsEmpty())
	require.True(t, excerpt.RedirectRef.IsEmpty())
	require.True(t, excerpt.RecordBodyHash.IsEmpty())
}

func TestExcerptReadPartial(t *testing.T) {
	rec := rms.ROutboundRequest{
		PrevRef:    rms.NewReference(gen.UniqueGlobalRef()),
		RootRef:    rms.NewReference(gen.UniqueGlobalRef()),
		Str:        "abc",
	}

	b, err := rec.Marshal()
	require.NoError(t, err)

	// Make the last field broken
	b = b[:len(b) - 1]

	// Ensure it is broken
	_, _, err = rmsreg.Unmarshal(b)
	require.Error(t, err)

	// Excerpt can read, because it only reads a subset of fields
	excerpt, err := ReadExcerptFromBytes(b)
	require.NoError(t, err)

	require.Equal(t, rec.GetDefaultPolymorphID(), uint64(excerpt.RecordType))
	require.Equal(t, rec.PrevRef, excerpt.PrevRef)
	require.Equal(t, rec.RootRef, excerpt.RootRef)
	require.True(t, excerpt.ReasonRef.IsEmpty())
	require.True(t, excerpt.RedirectRef.IsEmpty())
	require.True(t, excerpt.RecordBodyHash.IsEmpty())
}

