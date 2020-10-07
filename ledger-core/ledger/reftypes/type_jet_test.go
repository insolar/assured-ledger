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
	// Play with IDs
	// 0x03556677 - exactID which consist of
	// 0x03 - length of jetID (MSB of exactID)
	// 0x55 - not used
	// 0x6677 - jetID
	id := jet.LegID(pulse.MaxTimePulse | (0x03556677 << 32))
	assert.Equal(t, true, jet.LegID(id).ExactID().HasLength())

	// Play with refs
	jetLocalRef1 := JetLocalRef(0)
	jetLocalRef2 := JetLocalRef(77)
	jetLocalRef3 := JetLocalRef(77)
	assert.False(t, jetLocalRef1.Equal(jetLocalRef2))
	assert.True(t, jetLocalRef2.Equal(jetLocalRef3))
	assert.True(t, jetLocalRef3.Equal(jetLocalRef2))
	assert.Equal(t, "257/insolar:0AAABAQ.record", jetLocalRef1.String())
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

	id1 := jet.LegID(pulse.MaxTimePulse | (0x03006677 << 32))
	id2 := jet.LegID(pulse.MaxTimePulse | (0x03006677 << 32))
	id3 := jet.LegID(pulse.MaxTimePulse | (0x03006679 << 32))

	jetLegLocalRef1 := JetLegLocalRef(id1)
	jetLegLocalRef2 := JetLegLocalRef(id2)
	jetLegLocalRef3 := JetLegLocalRef(id3)
	assert.True(t, jetLegLocalRef1.Equal(jetLegLocalRef2))
	assert.False(t, jetLegLocalRef1.Equal(jetLegLocalRef3))
	assert.Equal(t, "257/insolar:0AAABAf___z93ZgAC.record", jetLegLocalRef1.String())
	assert.Equal(t, pulse.Jet, jetLegLocalRef1.GetPulseNumber())
	assert.Equal(t, jetLegLocalRef1.Pulse(), jetLegLocalRef1.GetPulseNumber())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x79, 0x66, 0, 2}, jetLegLocalRef3.GetHash())
}

