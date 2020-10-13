// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestRecordPayloadRef(t *testing.T) {
	var (
		jetID     = jet.ID(77)
		recordRef = gen.UniqueGlobalRef()
		offset    = uint32(3)
		length    = uint32(7)
		extID     = uint32(11)
	)
	recPayloadRef := RecordPayloadRef(jetID, recordRef, offset, length, extID)
	newID, newLocalRef, newOffset, newLength, newExtID, err := UnpackRecordPayloadRef(recPayloadRef)
	require.NoError(t, err)
	require.Equal(t, recordRef.GetLocal(), newLocalRef)
	require.Equal(t, jetID, newID)
	require.Equal(t, offset, newOffset)
	require.Equal(t, length, newLength)
	require.Equal(t, extID, newExtID)
}

func TestRecordPayloadRefByDrop(t *testing.T) {
	var (
		dropID    = jet.DropID(pulse.MinTimePulse)
		recordRef = gen.UniqueGlobalRefWithPulse(dropID.CreatedAt())
		offset    = uint32(3)
		length    = uint32(7)
		extID     = uint32(11)
	)

	recPayloadRef := RecordPayloadRefByDrop(dropID, recordRef, offset, length, extID)
	newID, newLocalRef, newOffset, newLength, newExtID, err := UnpackRecordPayloadRef(recPayloadRef)
	require.NoError(t, err)
	require.Equal(t, recordRef.GetLocal(), newLocalRef)
	require.Equal(t, dropID.ID(), newID)
	require.Equal(t, offset, newOffset)
	require.Equal(t, length, newLength)
	require.Equal(t, extID, newExtID)
}

func TestTypeDefRecPayload_RefFrom(t *testing.T) {
	var (
		jetID     = jet.ID(77)
		recordRef = gen.UniqueGlobalRef()
		offset    = uint32(3)
		length    = uint32(7)
		extID     = uint32(11)
	)
	recPayloadRef := RecordPayloadRef(jetID, recordRef, offset, length, extID)
	newRecord, err := tDefRecPayload.RefFrom(recPayloadRef.GetBase(), recPayloadRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, recPayloadRef, newRecord)
}

func TestRecordPayloadRef_BadInput(t *testing.T) {
	require.Panics(t, func() {
		RecordPayloadRefByDrop(jet.DropID(0), gen.UniqueGlobalRef(), 0, 1, 0)
	})

	require.Panics(t, func() {
		dropID := jet.DropID(pulse.MinTimePulse)
		RecordPayloadRefByDrop(dropID, gen.UniqueGlobalRefWithPulse(dropID.CreatedAt()+1), 0, 1, 0)
	})

	require.Panics(t, func() {
		dropID := jet.DropID(pulse.MinTimePulse)
		RecordPayloadRefByDrop(dropID, gen.UniqueGlobalRefWithPulse(dropID.CreatedAt()), 0, 0, 0)
	})

	require.Panics(t, func() {
		RecordPayloadRef(jet.ID(7), gen.UniqueGlobalRefWithPulse(1), 0, 0, 0)
	})
}
