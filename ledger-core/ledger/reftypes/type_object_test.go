// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestLifelineLocalRefOf(t *testing.T) {
	lifelineRef := LifelineLocalRefOf(gen.UniqueGlobalRef())
	newLocal, err := UnpackObjectLocalRef(lifelineRef)
	require.NoError(t, err)
	require.Equal(t, lifelineRef.GetLocal(), newLocal)
	err = tDefObject.VerifyLocalRef(lifelineRef.GetLocal())
	require.NoError(t, err)
}

func TestLifelineRef(t *testing.T) {
	localRef := gen.UniqueLocalRef()
	lifelineRef := LifelineRef(localRef)
	newBase, newLocal, err := UnpackObjectRef(lifelineRef)
	require.NoError(t, err)
	require.Equal(t, lifelineRef.GetLocal(), newLocal)
	require.Equal(t, lifelineRef.GetBase(), newBase)
	err = tDefObject.VerifyGlobalRef(lifelineRef.GetBase(), lifelineRef.GetLocal())
	require.NoError(t, err)
}

func TestLifelineRefOf(t *testing.T) {
	lifelineRef := LifelineRefOf(gen.UniqueGlobalRef())
	newBase, newLocal, err := UnpackObjectRef(lifelineRef)
	require.NoError(t, err)
	require.Equal(t, lifelineRef.GetLocal(), newLocal)
	require.Equal(t, lifelineRef.GetBase(), newBase)
	err = tDefObject.VerifyGlobalRef(lifelineRef.GetBase(), lifelineRef.GetLocal())
	require.NoError(t, err)
}

func TestSidelineRef(t *testing.T) {
	localRef := gen.UniqueLocalRef()
	lifelineRef := LifelineRef(localRef)
	sidelineRef := SidelineRef(lifelineRef, localRef)
	newBase, newLocal, err := UnpackObjectRef(lifelineRef)
	require.NoError(t, err)
	require.Equal(t, sidelineRef.GetLocal(), newLocal)
	require.Equal(t, sidelineRef.GetBase(), newBase)
	err = tDefObject.VerifyGlobalRef(sidelineRef.GetBase(), sidelineRef.GetLocal())
	require.NoError(t, err)
}

func TestRecordRef(t *testing.T) {
	localRef := gen.UniqueLocalRef()
	lifelineRef := LifelineRef(localRef)
	recordRef := RecordRef(lifelineRef, localRef)
	newBase, newLocal, err := UnpackObjectRef(lifelineRef)
	require.NoError(t, err)
	require.Equal(t, recordRef.GetLocal(), newLocal)
	require.Equal(t, recordRef.GetBase(), newBase)
	err = tDefObject.VerifyGlobalRef(recordRef.GetBase(), recordRef.GetLocal())
	require.NoError(t, err)
}

func TestTypeObject_RefFrom(t *testing.T) {
	lifelineRef := LifelineRef(gen.UniqueLocalRef())
	newRef, err := tDefObject.RefFrom(lifelineRef.GetBase(), lifelineRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, lifelineRef, newRef)
}

func TestTypeDefObject_CanBeDerivedWith(t *testing.T) {
	localRef := gen.UniqueLocalRef()
	require.True(t, tDefObject.CanBeDerivedWith(localRef.Pulse(), localRef))
	require.False(t, tDefObject.CanBeDerivedWith(localRef.Pulse()+1, localRef))
	require.False(t, tDefObject.CanBeDerivedWith(0, reference.Empty().GetLocal()))
}

func TestTypeObject_BadInput(t *testing.T) {
	require.Panics(t, func() {
		LifelineRef(reference.Empty().GetLocal())
	})

	require.Panics(t, func() {
		SidelineRef(reference.Empty(), reference.Local{})
	})

	require.Panics(t, func() {
		RecordRef(reference.Empty(), reference.Local{})
	})

	_, _, err := UnpackObjectRef(reference.Empty())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

	_, err = UnpackObjectLocalRef(reference.Empty().GetLocal())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())
}
