// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func TestNodeRefLocal(t *testing.T) {
	localNodeRef := NodeLocalRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	assert.Equal(t, pulse.Node, localNodeRef.Pulse())
	assert.EqualValues(t, 0, localNodeRef.SubScope()) // SubScope is not applicable and must be zero

	tDefNode = typeDefNode{}
	err := tDefNode.VerifyLocalRef(localNodeRef)
	assert.NoError(t, err)

	unpackedLocalNodeRef, err := UnpackNodeLocalRef(localNodeRef)
	assert.NoError(t, err)
	assert.Equal(t, localNodeRef, unpackedLocalNodeRef)

	pn, nrl := nodeLocalRefOf(localNodeRef)
	assert.Equal(t, pulse.Node, pn)
	assert.Equal(t, localNodeRef, nrl)

	callLocalNodeRef := APICallRef(reference.NewSelf(localNodeRef), pulse.MinTimePulse, reference.LocalHash{})

	pn, nrl = nodeLocalRefOf(callLocalNodeRef.GetBase())
	assert.Equal(t, pulse.ExternalCall, pn)
	assert.Equal(t, localNodeRef, nrl)

	localNodeRef1 := NodeLocalRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	assert.Equal(t, localNodeRef, localNodeRef1)
}

func TestNodeRef(t *testing.T) {
	nodeRef := NodeRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})

	assert.Equal(t, pulse.Node, nodeRef.GetBase().GetPulseNumber())
	assert.EqualValues(t, 0, nodeRef.GetBase().SubScope()) // SubScope is not applicable and must be zero
	assert.Equal(t, nodeRef.GetBase(), nodeRef.GetLocal())

	tDefNode = typeDefNode{}
	err := tDefNode.VerifyGlobalRef(nodeRef.GetBase(), nodeRef.GetLocal())
	assert.NoError(t, err)

	err = tDefNode.VerifyLocalRef(nodeRef.GetLocal())
	assert.NoError(t, err)

	unpackedLocalNodeRef, err := UnpackNodeLocalRef(nodeRef)
	assert.NoError(t, err)
	assert.Equal(t, nodeRef.GetLocal(), unpackedLocalNodeRef)

	unpackedNodeRef, err := UnpackNodeRef(nodeRef)
	assert.NoError(t, err)
	assert.Equal(t, nodeRef, unpackedNodeRef)

	unpackedNodeRefAsLocal, err := UnpackNodeRefAsLocal(nodeRef)
	assert.Equal(t, nodeRef.GetLocal(), unpackedNodeRefAsLocal)

	_, _, err = UnpackNodeContractRef(nodeRef)
	assert.Error(t, err)

	require.True(t, tDefNode.CanBeDerivedWith(0, reference.NewLocal(pulse.MinTimePulse, 0, reference.LocalHash{})))
	require.False(t, tDefNode.CanBeDerivedWith(0, reference.Empty().GetLocal()))
}

func TestNodeContractRef(t *testing.T) {
	nodeRef := NodeRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})

	contractLoc := reference.NewLocal(pulse.MinTimePulse, reference.SubScopeLocal, reference.LocalHash{})
	nodeContractRef := NodeContractRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, contractLoc)

	require.Equal(t, nodeRef.GetBase(), nodeContractRef.GetBase())
	require.Equal(t, contractLoc, nodeContractRef.GetLocal())

	_, _, err := UnpackNodeContractRef(nodeRef)
	assert.Error(t, err)

	unpackedNodeRef, unpackedContractRef, err := UnpackNodeContractRef(nodeContractRef)
	assert.NoError(t, err)
	require.Equal(t, nodeRef, unpackedNodeRef)
	require.Equal(t, contractLoc, unpackedContractRef)
}

func TestNodeRefDetect(t *testing.T) {
	nodeRef := NodeRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	assert.Equal(t, Node, tDefNode.DetectSubType(nodeRef.GetBase(), nodeRef.GetLocal()))

	contractLoc := reference.NewLocal(pulse.MinTimePulse, reference.SubScopeLocal, reference.LocalHash{})
	nodeContractRef := NodeContractRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, contractLoc)
	assert.Equal(t, NodeContract, tDefNode.DetectSubType(nodeContractRef.GetBase(), nodeContractRef.GetLocal()))
}

func TestTypeNode_RefFrom(t *testing.T) {
	nodeRef := NodeRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	newNodeRef, err := tDefNode.RefFrom(nodeRef.GetBase(), nodeRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, nodeRef, newNodeRef)

	tDefNodeContract.RefFrom(nodeRef.GetBase(), nodeRef.GetLocal())
	require.NoError(t, err)
	require.Equal(t, nodeRef, newNodeRef)
}
