// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gen

import (
	"encoding/binary"
	"sync/atomic"

	fuzz "github.com/google/gofuzz"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

var uniqueSeq uint32

func getUnique() uint32 {
	return atomic.AddUint32(&uniqueSeq, 1)
}

// ID generates random id.
func ID() insolar.ID {
	var id insolar.ID

	f := fuzz.New().NilChance(0).Funcs(func(id *insolar.ID, c fuzz.Continue) {
		var hash [reference.LocalBinaryHashSize]byte
		c.Fuzz(&hash)
		binary.BigEndian.PutUint32(hash[reference.LocalBinaryHashSize-4:], getUnique())

		pn := PulseNumber()

		*id = insolar.NewID(pn, hash[:])
	})
	f.Fuzz(&id)

	return id
}

// IDWithPulse generates random id with provided pulse.
func IDWithPulse(pn insolar.PulseNumber) insolar.ID {
	hash := make([]byte, reference.LocalBinaryHashSize)

	fuzz.New().
		NilChance(0).
		NumElements(insolar.RecordHashSize, insolar.RecordHashSize).
		Fuzz(&hash)
	return insolar.NewID(pn, hash)
}

// Reference generates random reference.
func Reference() insolar.Reference {
	return insolar.NewReference(ID())
}

// UniqueReferences generates multiple random unique References.
func UniqueReferences(a int) []insolar.Reference {
	refs := make([]insolar.Reference, a)

	for i := 0; i < a; i++ {
		refs[i] = Reference()
	}
	return refs
}
