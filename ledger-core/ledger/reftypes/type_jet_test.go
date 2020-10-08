// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func TestJetLocalRef(t *testing.T) {
	jetLocalRef1 := JetLocalRef(0)
	assert.Equal(t, pulse.Jet, jetLocalRef1.Pulse())
	assert.Equal(t, reference.LocalHash{0}, jetLocalRef1.IdentityHash())

	jetLocalRef2 := JetLocalRef(77)
	assert.Equal(t, pulse.Jet, jetLocalRef2.Pulse())
	assert.Equal(t, reference.LocalHash{0, 0, 0, 0, 77}, jetLocalRef2.IdentityHash())

	tDefJet := typeDefJet{}
	err := tDefJet.VerifyLocalRef(jetLocalRef1)
	assert.NoError(t, err)

	err = tDefJet.VerifyLocalRef(jetLocalRef2)
	assert.NoError(t, err)

	tp := tDefJet.detectJetSubType(jetLocalRef1, false)
	assert.Equal(t, Jet, tp)
	tp = tDefJet.detectJetSubType(jetLocalRef1, true)
	assert.Equal(t, Invalid, tp)

	jetId, err := UnpackJetLocalRef(jetLocalRef2)
	assert.NoError(t, err)
	assert.Equal(t, jet.ID(77), jetId)

	jetId, err = unpackJetLocalRef(jetLocalRef2, true)
	assert.NoError(t, err)
	assert.Equal(t, jet.ID(77), jetId)

	hash := jetLocalRef2.IdentityHash()
	hash[len(hash) - 1] = 1
	jetLocalRef3 := jetLocalRef2.WithHash(hash)

	jetId, err = unpackJetLocalRef(jetLocalRef3, true)
	assert.Error(t, err)

	jetId, err = unpackJetLocalRef(jetLocalRef3, false)
	assert.NoError(t, err)
	assert.Equal(t, jet.ID(77), jetId)
}

func TestJetLegLocalRef(t *testing.T) {
	// Play with IDs
	// 0x11006677 - exactID which consist of
	// 0x11 - bit length of jetID (MSB of exactID) minus one
	// 0x00 - reserved, must be 0
	// 0x6677 - jetID

	const prefixLen = 16
	id1 := jet.ID(0x6677).AsLeg(prefixLen, pulse.MaxTimePulse)
	id2 := jet.ID(0x6679).AsLeg(prefixLen, pulse.MaxTimePulse)

	jetLegLocalRef1 := JetLegLocalRef(id1)
	jetLegLocalRef3 := JetLegLocalRef(id2)
	assert.NotEqual(t, jetLegLocalRef1, jetLegLocalRef3)

	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x77, 0x66, 0, prefixLen + 1}, jetLegLocalRef1.IdentityHash())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x79, 0x66, 0, prefixLen + 1}, jetLegLocalRef3.IdentityHash())
	assert.EqualValues(t, 0, jetLegLocalRef1.SubScope()) // SubScope is not applicable and must be zero

	pn, exactID, err := DecodeJetData(jetLegLocalRef1.IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, id1.ExactID(), exactID)

	pn, exactID, err = DecodeJetData(jetLegLocalRef3.IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, id2.ExactID(), exactID)

	unpackedJetID, err := UnpackJetLegLocalRef(jetLegLocalRef1)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id1)

	unpackedJetID, err = UnpackJetLegLocalRef(jetLegLocalRef3)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id2)
}

