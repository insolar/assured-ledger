// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gen

import (
	"encoding/binary"
	"sync/atomic"

	fuzz "github.com/google/gofuzz"

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

// UniqueID generates random unique id.
func UniqueID() reference.Local {
	return generateUniqueID(PulseNumber())
}

func UniqueIDWithPulse(pn pulse.Number) reference.Local {
	return generateUniqueID(pn)
}

// UniqueReference generates random reference.
func UniqueReference() reference.Global {
	id := UniqueID()
	return reference.NewSelf(id)
}

// UniqueReferences generates multiple random unique References.
func UniqueReferences(n int) []reference.Global {
	refs := make([]reference.Global, n)

	for i := 0; i < n; i++ {
		refs[i] = UniqueReference()
	}
	return refs
}
