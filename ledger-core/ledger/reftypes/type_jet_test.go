// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJetLocalRef(t *testing.T) {
	jetLocalRef1 := JetLocalRef(0)
	jetLocalRef2 := JetLocalRef(77)
	jetLocalRef3 := JetLocalRef(77)
	assert.False(t, jetLocalRef1.Equal(jetLocalRef2))
	assert.True(t, jetLocalRef2.Equal(jetLocalRef3))
	assert.True(t, jetLocalRef3.Equal(jetLocalRef2))
	assert.Equal(t, pulse.Jet, jetLocalRef1.GetPulseNumber())
	assert.Equal(t, jetLocalRef1.Pulse(), jetLocalRef1.GetPulseNumber())
	assert.Equal(t, reference.LocalHash{0, 0, 0, 0, 77}, jetLocalRef3.GetHash())
	assert.Equal(t, reference.LocalHash{0}, jetLocalRef1.GetHash())

	tDefJet := typeDefJet{}
	err := tDefJet.VerifyLocalRef(jetLocalRef1)
	assert.Equal(t, nil, err)

	tp := tDefJet.detectJetSubType(jetLocalRef1, false)
	assert.Equal(t, Jet, tp)
	tp = tDefJet.detectJetSubType(jetLocalRef1, true)
	assert.Equal(t, Invalid, tp)

	jetId, err := UnpackJetLocalRef(jetLocalRef3)
	assert.Equal(t, jet.ID(77), jetId)
	assert.Equal(t, nil, err)

	jetId, err = PartialUnpackJetLocalRef(jetLocalRef3)
	assert.Equal(t, jet.ID(77), jetId)
	assert.Equal(t, nil, err)

	jetId, err = unpackJetLocalRef(jetLocalRef3, true)
	assert.Equal(t, jet.ID(77), jetId)
	assert.Equal(t, nil, err)

	jetId, err = unpackJetLocalRef(jetLocalRef3, false)
	assert.Equal(t, jet.ID(77), jetId)
	assert.Equal(t, nil, err)
}

func TestJetLegLocalRef(t *testing.T) {
	// Play with IDs
	// 0x11006677 - exactID which consist of
	// 0x11 - bit length of jetID (MSB of exactID) minus one
	// 0x00 - reserved, must be 0
	// 0x6677 - jetID
	id := jet.LegID(pulse.MaxTimePulse | (0x11006677 << 32))
	assert.Equal(t, true, jet.LegID(id).ExactID().HasLength())

	id1 := jet.LegID(pulse.MaxTimePulse | (0x11006677 << 32))
	id2 := jet.LegID(pulse.MaxTimePulse | (0x11006677 << 32))
	id3 := jet.LegID(pulse.MaxTimePulse | (0x11006679 << 32))

	jetLegLocalRef1 := JetLegLocalRef(id1)
	jetLegLocalRef2 := JetLegLocalRef(id2)
	jetLegLocalRef3 := JetLegLocalRef(id3)
	assert.True(t, jetLegLocalRef1.Equal(jetLegLocalRef2))
	assert.False(t, jetLegLocalRef1.Equal(jetLegLocalRef3))
	assert.Equal(t, pulse.Jet, jetLegLocalRef1.GetPulseNumber())
	assert.Equal(t, jetLegLocalRef1.Pulse(), jetLegLocalRef1.GetPulseNumber())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x79, 0x66, 0, 0x10}, jetLegLocalRef3.GetHash())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x79, 0x66, 0, 0x10}, jetLegLocalRef3.IdentityHash())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x77, 0x66, 0, 0x10}, jetLegLocalRef1.GetHash())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x77, 0x66, 0, 0x10}, jetLegLocalRef1.IdentityHash())
	assert.Equal(t, reference.SubScopeLifeline, jetLegLocalRef1.SubScope())

	pn, exactID, err := DecodeJetData(jetLegLocalRef1.IdentityHash(), true)

	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, jet.ExactID(0x11006677), exactID)
	assert.Nil(t, err)

	unpackedJetID, err := UnpackJetLegLocalRef(jetLegLocalRef1)
	assert.Equal(t, unpackedJetID, id1)
	assert.Nil(t, err)

}

