// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func TestBuiltinContractLocalRef(t *testing.T) {
	hash := BuiltinContractID{1, 2, 3, 4}
	localRef := BuiltinContractLocalRef(ExternalNodeAPI, hash)
	require.Equal(t, pulse.BuiltinContract, localRef.Pulse())
	require.Equal(t, byte(ExternalNodeAPI), localRef.IdentityHashBytes()[0])
	require.Equal(t, hash[:], localRef.IdentityHashBytes()[1:])
}

func TestUnpackBuiltinContractRef(t *testing.T) {
	hash := BuiltinContractID{1, 2, 3, 4}
	builtinRef := BuiltinContractRef(ExternalNodeAPI, hash)
	refType, primaryID, secondaryID, err := UnpackBuiltinContractRef(builtinRef)
	require.NoError(t, err)
	require.Equal(t, ExternalNodeAPI, refType)
	require.Nil(t, secondaryID)
	require.Equal(t, builtinRef.GetLocal().IdentityHashBytes()[1:], primaryID[:])

	hash2 := BuiltinContractID{5, 6, 7, 8}
	localBuiltinRef1 := BuiltinContractLocalRef(ExternalNodeAPI, hash)
	localBuiltinRef2 := BuiltinContractLocalRef(ExternalNodeAPI, hash2)
	refType, primaryID, secondaryID, err = UnpackBuiltinContractRef(reference.New(localBuiltinRef1, localBuiltinRef2))
	require.NoError(t, err)
	require.Equal(t, ExternalNodeAPI, refType)
	require.Equal(t, localBuiltinRef1.IdentityHashBytes()[1:], primaryID[:])
	require.Equal(t, localBuiltinRef2, secondaryID)
}

func TestUnpackBuiltinContractRef_BadInput(t *testing.T) {
	jetRef := JetRef(0)
	_, _, _, err := UnpackBuiltinContractRef(jetRef)
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

	hash := BuiltinContractID{5, 6, 7, 8}

	_, _, _, err = UnpackBuiltinContractRef(reference.New(BuiltinContractLocalRef(ExternalNodeAPI, hash), reference.Empty().GetLocal()))
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

}

func TestTypeDefBuiltinContract_RefFrom(t *testing.T) {
	hash := BuiltinContractID{1, 2, 3, 4}
	builtinRef := BuiltinContractRef(ExternalNodeAPI, hash)
	newRef, err := tDefBuiltinContract.RefFrom(builtinRef.GetBase(), builtinRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, builtinRef, newRef)

	// bad input
	jetRef := JetRef(0)
	_, err = tDefBuiltinContract.RefFrom(jetRef.GetBase(), jetRef.GetLocal())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())
}
