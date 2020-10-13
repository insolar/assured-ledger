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

func TestAPICallRef(t *testing.T) {
	localNodeRef := NodeLocalRef(reference.LocalHash{1, 2, 3, 4})
	reqHash := reference.LocalHash{5, 6, 7, 8}
	sidPN := pulse.MinTimePulse + pulse.Number(300)
	apiCallRef := APICallRef(reference.NewSelf(localNodeRef), sidPN, reqHash)
	unpackedNodeRef, unpackedSidPN, unpackedReqHash, err := UnpackAPICallRef(apiCallRef)

	require.NoError(t, err)
	require.Equal(t, reqHash, unpackedReqHash)
	require.Equal(t, sidPN, unpackedSidPN)
	require.Equal(t, reference.NewSelf(localNodeRef), unpackedNodeRef)
}

func TestAPICallRef_BadInput(t *testing.T) {
	require.Panics(t, func() {
		_ = APICallRef(reference.Empty(), pulse.Number(1), reference.LocalHash{})
	})
	require.Panics(t, func() {
		_ = APICallRef(reference.Empty(), pulse.MinTimePulse, reference.LocalHash{})
	})
}

func TestUnpackAPICallRef_BadInput(t *testing.T) {
	_, _, _, err := UnpackAPICallRef(reference.Empty())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())

	localRef := reference.NewLocal(pulse.MinTimePulse+pulse.Number(1), 0, reference.LocalHash{})
	_, _, _, err = UnpackAPICallRef(reference.NewSelf(localRef))
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())
}

func TestTypeDefAPICall_RefFrom(t *testing.T) {
	localNodeRef := NodeLocalRef(reference.LocalHash{1, 2, 3, 4})
	reqHash := reference.LocalHash{5, 6, 7, 8}
	sidPN := pulse.MinTimePulse + pulse.Number(300)
	apiCallRef := APICallRef(reference.NewSelf(localNodeRef), sidPN, reqHash)
	newRef, err := tDefAPICall.RefFrom(apiCallRef.GetBase(), apiCallRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, apiCallRef, newRef)

	// bad input
	_, err = tDefAPICall.RefFrom(reference.Empty().GetBase(), reference.Empty().GetLocal())
	require.Contains(t, err.Error(), ErrIllegalRefValue.Error())
}
