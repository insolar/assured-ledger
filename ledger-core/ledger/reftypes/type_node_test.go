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
}
