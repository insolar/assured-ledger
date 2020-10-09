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

	jetId, err = PartialUnpackJetLocalRef(jetLocalRef1)
	assert.NoError(t, err)
	assert.Equal(t, jet.ID(0), jetId)

	jetId, err = PartialUnpackJetLocalRef(jetLocalRef2)
	assert.NoError(t, err)
	assert.Equal(t, jet.ID(77), jetId)

}

func TestJetRef(t *testing.T) {
	jetRef1 := JetRef(0)
	assert.Equal(t, pulse.Jet, jetRef1.GetLocal().Pulse())
	assert.Equal(t, reference.LocalHash{0}, jetRef1.GetLocal().IdentityHash())
	assert.Equal(t, reference.Scope(0), jetRef1.GetScope()) // SubScope is not applicable and must be zero
	assert.True(t, jetRef1.IsLifelineScope())
	assert.False(t, jetRef1.IsGlobalScope())
	assert.False(t, jetRef1.IsLocalDomainScope())

	jetRef2 := JetRef(77)
	assert.Equal(t, pulse.Jet, jetRef2.GetLocal().Pulse())
	assert.Equal(t, reference.LocalHash{0, 0, 0, 0, 77}, jetRef2.GetLocal().IdentityHash())

	tDefJet := typeDefJet{}
	err := tDefJet.VerifyGlobalRef(jetRef1.GetBase(), jetRef1.GetLocal())
	assert.NoError(t, err)

	err = tDefJet.VerifyLocalRef(jetRef1.GetLocal())
	assert.NoError(t, err)

	err = tDefJet.VerifyLocalRef(JetLocalRefOf(jetRef1))
	assert.NoError(t, err)

	tp := tDefJet.DetectSubType(jetRef1.GetBase(), jetRef1.GetLocal())
	assert.Equal(t, Jet, tp)

	jetId, err := UnpackJetRef(jetRef2)
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
	jetLegLocalRef2 := JetLegLocalRef(id2)
	assert.NotEqual(t, jetLegLocalRef1, jetLegLocalRef2)

	tDefJetLeg := typeDefJetLeg{}
	err := tDefJetLeg.VerifyLocalRef(jetLegLocalRef1)
	assert.NoError(t, err)

	err = tDefJetLeg.VerifyLocalRef(jetLegLocalRef2)
	assert.NoError(t, err)

	tp := tDefJet.detectJetSubType(jetLegLocalRef1, false)
	assert.Equal(t, JetLeg, tp)
	tp = tDefJet.detectJetSubType(jetLegLocalRef2, true)
	assert.Equal(t, Invalid, tp)

	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x77, 0x66, 0, prefixLen + 1}, jetLegLocalRef1.IdentityHash())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x79, 0x66, 0, prefixLen + 1}, jetLegLocalRef2.IdentityHash())
	assert.EqualValues(t, 0, jetLegLocalRef1.SubScope()) // SubScope is not applicable and must be zero

	pn, exactID, err := DecodeJetData(jetLegLocalRef1.IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, id1.ExactID(), exactID)

	pn, exactID, err = DecodeJetData(jetLegLocalRef2.IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, id2.ExactID(), exactID)

	unpackedJetID, err := UnpackJetLegLocalRef(jetLegLocalRef1)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id1)

	unpackedJetID, err = UnpackJetLegLocalRef(jetLegLocalRef2)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id2)
}

func TestJetLegRef(t *testing.T) {
	// Play with IDs
	// 0x11006677 - exactID which consist of
	// 0x11 - bit length of jetID (MSB of exactID) minus one
	// 0x00 - reserved, must be 0
	// 0x6677 - jetID

	const prefixLen = 16
	id1 := jet.ID(0x6677).AsLeg(prefixLen, pulse.MaxTimePulse)
	id2 := jet.ID(0x6679).AsLeg(prefixLen, pulse.MaxTimePulse)

	jetLegRef1 := JetLegRef(id1)
	jetLegRef2 := JetLegRef(id2)
	assert.NotEqual(t, jetLegRef1, jetLegRef2)

	tDefJetLeg := typeDefJetLeg{}
	err := tDefJetLeg.VerifyGlobalRef(jetLegRef1.GetBase(), jetLegRef1.GetLocal())
	assert.NoError(t, err)

	err = tDefJetLeg.VerifyGlobalRef(jetLegRef2.GetBase(), jetLegRef2.GetLocal())
	assert.NoError(t, err)

	tp := tDefJet.DetectSubType(jetLegRef1.GetBase(), jetLegRef1.GetLocal())
	assert.Equal(t, JetLeg, tp)

	unpackedJetID, err := UnpackJetLegRef(jetLegRef1)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id1)

	unpackedJetID, err = UnpackJetLegRef(jetLegRef2)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id2)
}

func TestJetDropLocalRef(t *testing.T) {

	id1 := jet.ID(0x6677).AsDrop(pulse.MaxTimePulse)
	id2 := jet.ID(0x6679).AsDrop(pulse.MaxTimePulse)

	jetDropLocalRef1 := JetDropLocalRef(id1)
	jetDropLocalRef2 := JetDropLocalRef(id2)
	assert.NotEqual(t, jetDropLocalRef1, jetDropLocalRef2)


	tDefJetDrop := typeDefJetDrop{}
	err := tDefJetDrop.VerifyLocalRef(jetDropLocalRef1)
	assert.NoError(t, err)

	err = tDefJetDrop.VerifyLocalRef(jetDropLocalRef2)
	assert.NoError(t, err)

	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x77, 0x66}, jetDropLocalRef1.IdentityHash())
	assert.Equal(t, reference.LocalHash{0xFF, 0xFF, 0xFF, 0x3F, 0x79, 0x66}, jetDropLocalRef2.IdentityHash())
	assert.EqualValues(t, 0, jetDropLocalRef1.SubScope()) // SubScope is not applicable and must be zero

	pn, id, err := DecodeJetData(jetDropLocalRef1.IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, jet.ExactID(0x6677), id)

	pn, id, err = DecodeJetData(jetDropLocalRef2.IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, jet.ExactID(0x6679), id)

	unpackedJetID, err := UnpackJetDropLocalRef(jetDropLocalRef1)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id1)

	unpackedJetID, err = UnpackJetDropLocalRef(jetDropLocalRef2)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id2)
}


func TestJetDropRef(t *testing.T) {

	id1 := jet.ID(0x6677).AsDrop(pulse.MaxTimePulse)
	id2 := jet.ID(0x6679).AsDrop(pulse.MaxTimePulse)

	jetDropRef1 := JetDropRef(id1)
	jetDropRef2 := JetDropRef(id2)
	assert.NotEqual(t, jetDropRef1, jetDropRef2)

	tDefJetDrop := typeDefJetDrop{}
	err := tDefJetDrop.VerifyGlobalRef(jetDropRef1.GetBase(), jetDropRef1.GetLocal())
	assert.NoError(t, err)

	tDefJetDrop = typeDefJetDrop{}
	err = tDefJetDrop.VerifyLocalRef(jetDropRef1.GetLocal())
	assert.NoError(t, err)

	err = tDefJetDrop.VerifyLocalRef(JetDropLocalRefOf(jetDropRef1))
	assert.NoError(t, err)

	unpackedJetID, err := UnpackJetDropRef(jetDropRef1)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id1)

	unpackedJetID, err = UnpackJetDropRef(jetDropRef2)
	assert.NoError(t, err)
	assert.Equal(t, unpackedJetID, id2)

	pn, exactID, err := DecodeJetData(jetDropRef1.GetLocal().IdentityHash(), true)
	assert.NoError(t, err)
	assert.Equal(t, pulse.Number(pulse.MaxTimePulse), pn)
	assert.Equal(t, jet.ExactID(0x6677), exactID)
}
