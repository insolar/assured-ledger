// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestLocalNodeRef(t *testing.T) {
	localNodeRef := NodeLocalRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	assert.Equal(t, "260/insolar:0AAABBN6tvu8.record", localNodeRef.String())

	tDefNode = typeDefNode{}
	err := tDefNode.VerifyLocalRef(localNodeRef)
	assert.Equal(t, nil, err)

	unpackedLocalNodeRef, err := UnpackNodeLocalRef(localNodeRef)
	assert.Equal(t, nil, err)
	assert.Equal(t, pulse.Node, unpackedLocalNodeRef.GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, unpackedLocalNodeRef.SubScope())
	assert.True(t, reflect.DeepEqual(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, unpackedLocalNodeRef.GetHash()))

	pn, nrl := nodeLocalRefOf(localNodeRef)
	assert.Equal(t, pulse.Node, pn)
	assert.Equal(t, "260/insolar:0AAABBN6tvu8.record", nrl.String())
	assert.Equal(t, pulse.Node, nrl.GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, nrl.SubScope())
	assert.True(t, reflect.DeepEqual(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, nrl.GetHash()))

	localNodeRef1 := NodeLocalRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	assert.True(t, localNodeRef.Equal(localNodeRef1))
}


func TestGlobalNodeRef(t *testing.T) {
	nodeRef := NodeRef(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF})
	assert.Equal(t, "insolar:0AAABBN6tvu8", nodeRef.String())
	assert.Equal(t, "260/insolar:0AAABBN6tvu8.record", nodeRef.GetBase().String())
	assert.Equal(t, "260/insolar:0AAABBN6tvu8.record", nodeRef.GetLocal().String())
	assert.Equal(t, pulse.Node, nodeRef.GetBase().GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, nodeRef.GetBase().SubScope())
	assert.Equal(t, pulse.Node, nodeRef.GetLocal().GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, nodeRef.GetLocal().SubScope())

	tDefNode = typeDefNode{}
	err := tDefNode.VerifyGlobalRef(nodeRef.GetBase(), nodeRef.GetLocal())
	assert.Equal(t, nil, err)
	err = tDefNode.VerifyLocalRef(nodeRef.GetLocal())
	assert.Equal(t, nil, err)
	assert.Equal(t, Node, tDefNode.DetectSubType(nodeRef.GetBase(), nodeRef.GetLocal()))

	unpackedLocalNodeRef, err := UnpackNodeLocalRef(nodeRef)
	assert.Equal(t, nil, err)
	assert.Equal(t, pulse.Node, unpackedLocalNodeRef.GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, unpackedLocalNodeRef.SubScope())
	assert.True(t, reflect.DeepEqual(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, unpackedLocalNodeRef.GetHash()))
	assert.True(t, unpackedLocalNodeRef.Equal(nodeRef.GetLocal()))

	unpackedNodeRef, err := UnpackNodeRef(nodeRef)
	assert.Equal(t, nil, err)
	assert.Equal(t, pulse.Node, unpackedNodeRef.GetLocal().GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, unpackedNodeRef.GetLocal().SubScope())
	assert.True(t, reflect.DeepEqual(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, unpackedNodeRef.GetLocal().GetHash()))
	assert.Equal(t, pulse.Node, unpackedNodeRef.GetBase().GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, unpackedNodeRef.GetBase().SubScope())
	assert.True(t, reflect.DeepEqual(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, unpackedNodeRef.GetBase().GetHash()))
	assert.Equal(t, "260/insolar:0AAABBN6tvu8.record", unpackedNodeRef.GetBase().String())
	assert.True(t, unpackedNodeRef.GetLocal().Equal(nodeRef.GetLocal()))
	assert.True(t, unpackedNodeRef.GetBase().Equal(nodeRef.GetBase()))


	unpackedNodeRefAsLocal, err := UnpackNodeRefAsLocal(nodeRef)
	assert.Equal(t, nil, err)
	assert.Equal(t, pulse.Node, unpackedNodeRefAsLocal.GetLocal().GetPulseNumber())
	assert.Equal(t, reference.SubScopeLifeline, unpackedNodeRefAsLocal.GetLocal().SubScope())
	assert.True(t, reflect.DeepEqual(reference.LocalHash{0xDE, 0xAD, 0xBE, 0xEF}, unpackedNodeRefAsLocal.GetLocal().GetHash()))
	assert.True(t, unpackedNodeRefAsLocal.GetLocal().Equal(nodeRef.GetLocal()))

	_, _, err = UnpackNodeContractRef(nodeRef)
	assert.NotNil(t, err)

}
