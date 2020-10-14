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
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestJetRecordRef(t *testing.T) {
	var (
		dropID = jet.ID(0x6677).AsDrop(pulse.MaxTimePulse)
		ref    = gen.UniqueGlobalRefWithPulse(dropID.CreatedAt())
	)
	{
		jetRecordRef := JetRecordRef(dropID.ID(), ref)
		newDropID, newRef, err := UnpackJetRecordRef(jetRecordRef)
		require.NoError(t, err)
		require.Equal(t, dropID, newDropID)
		require.Equal(t, ref.GetLocal(), newRef)
	}
	{
		jetRecordRef := JetRecordRefByDrop(dropID, ref)
		newDropID, newRef, err := UnpackJetRecordRef(jetRecordRef)
		require.NoError(t, err)
		require.Equal(t, dropID, newDropID)
		require.Equal(t, ref.GetLocal(), newRef)
	}
}

func TestJetRecordOf(t *testing.T) {
	var (
		dropID = jet.ID(0xffff).AsDrop(pulse.MaxTimePulse)
		ref    = gen.UniqueGlobalRefWithPulse(dropID.CreatedAt())
	)

	jetRecordRef := JetRecordOf(JetRecordRef(dropID.ID(), ref))

	require.NotEqual(t, jetRecordRef, reference.Global{})
	newDropID, newLocalRef, err := UnpackAsJetRecordOf(jetRecordRef)
	require.NoError(t, err)
	require.Equal(t, ref.GetLocal(), newLocalRef)
	require.Equal(t, dropID, newDropID)
}

func TestTypeJetRecord_RefFrom(t *testing.T) {
	var (
		dropID = jet.ID(0x6677).AsDrop(pulse.MaxTimePulse)
		ref    = gen.UniqueGlobalRefWithPulse(dropID.CreatedAt())
	)
	jetRecordRef := JetRecordRef(dropID.ID(), ref)
	newRef, err := tDefJetRecord.RefFrom(jetRecordRef.GetBase(), jetRecordRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, jetRecordRef, newRef)
}

func TestTypeJetRecord_BadInput(t *testing.T) {
	_, err := tDefJetRecord.RefFrom(reference.Empty().GetLocal(), reference.Empty().GetLocal())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

	_, _, err = UnpackJetRecordRef(nil)
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

	_, _, err = UnpackAsJetRecordOf(reference.Empty())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

	ref := JetRecordOf(nil)
	require.Equal(t, reference.Global{}, ref)

	require.Panics(t, func() {
		JetRecordRef(jet.ID(0), gen.UniqueGlobalRef())
	})

	require.Panics(t, func() {
		JetRecordRef(jet.ID(1), reference.Empty())
	})

	require.Panics(t, func() {
		JetRecordRefByDrop(jet.DropID(1), reference.Empty())
	})
}
