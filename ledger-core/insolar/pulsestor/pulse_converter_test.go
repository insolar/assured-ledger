// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func generatePsc() *SenderConfirmation {
	return &SenderConfirmation{
		PulseNumber:     32,
		ChosenPublicKey: "124",
		Entropy:         Entropy{123},
		Signature:       []byte("456"),
	}
}

func TestPulseSenderConfirmationToProto(t *testing.T) {
	p := generatePsc()
	proto := SenderConfirmationToProto("112", *p)
	pk, p2 := SenderConfirmationFromProto(proto)
	assert.Equal(t, "112", pk)
	assert.EqualValues(t, p.PulseNumber, p2.PulseNumber)
	assert.Equal(t, p.ChosenPublicKey, p2.ChosenPublicKey)
	assert.Equal(t, p.Entropy, p2.Entropy)
	assert.Equal(t, p.Signature, p2.Signature)
}
