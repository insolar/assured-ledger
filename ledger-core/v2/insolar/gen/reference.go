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
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

var uniqueSeq uint32

func getUnique() uint32 {
	return atomic.AddUint32(&uniqueSeq, 1)
}

func generateUniqueID(pn pulse.Number) reference.Local {
	var id reference.Local

	f := fuzz.New().NilChance(0).Funcs(func(id *reference.Local, c fuzz.Continue) {
		var hash [reference.LocalBinaryHashSize]byte
		c.Fuzz(&hash)
		binary.BigEndian.PutUint32(hash[reference.LocalBinaryHashSize-4:], getUnique())

		*id = reference.NewRecordID(pn, reference.BytesToLocalHash(hash[:]))
	})
	f.Fuzz(&id)

	return id
}

// ID generates random id.
func ID() reference.Local {
	return generateUniqueID(PulseNumber())
}

func UniqueIDWithPulse(pn insolar.PulseNumber) reference.Local {
	return generateUniqueID(pn)

}

// IDWithPulse generates random id with provided pulse.
func IDWithPulse(pn insolar.PulseNumber) reference.Local {
	hash := make([]byte, reference.LocalBinaryHashSize)

	fuzz.New().
		NilChance(0).
		NumElements(reference.LocalBinaryHashSize, reference.LocalBinaryHashSize).
		Fuzz(&hash)
	return reference.NewRecordID(pn, reference.BytesToLocalHash(hash))
}

// Reference generates random reference.
func Reference() reference.Global {
	id := ID()
	return reference.NewSelf(id)
}

// UniqueReferences generates multiple random unique References.
func UniqueReferences(a int) []reference.Global {
	refs := make([]reference.Global, a)

	for i := 0; i < a; i++ {
		refs[i] = Reference()
	}
	return refs
}
