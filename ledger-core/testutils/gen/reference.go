package gen

import (
	"encoding/binary"
	"sync/atomic"

	fuzz "github.com/google/gofuzz"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

var uniqueSeq uint32

func getUnique() uint32 {
	return atomic.AddUint32(&uniqueSeq, 1)
}

func generateUniqueLocalRef(pn pulse.Number) reference.Local {
	var local reference.Local

	f := fuzz.New().NilChance(0).Funcs(func(id *reference.Local, c fuzz.Continue) {
		var hash [reference.LocalBinaryHashSize]byte
		c.Fuzz(&hash)
		binary.BigEndian.PutUint32(hash[reference.LocalBinaryHashSize-4:], getUnique())

		*id = reference.NewRecordID(pn, reference.BytesToLocalHash(hash[:]))
	})
	f.Fuzz(&local)

	return local
}

// UniqueLocalRef generates a random unique reference.Local.
func UniqueLocalRef() reference.Local {
	return generateUniqueLocalRef(PulseNumber())
}

// UniqueLocalRefWithPulse generates a random unique reference.Local with a given pulse.Number.
func UniqueLocalRefWithPulse(pn pulse.Number) reference.Local {
	return generateUniqueLocalRef(pn)
}

// UniqueGlobalRef generates a random unique reference.Global.
func UniqueGlobalRef() reference.Global {
	local := UniqueLocalRef()
	return reference.NewSelf(local)
}

// UniqueGlobalRefWithPulse generates a random unique reference.Global with a given pulse.Number.
func UniqueGlobalRefWithPulse(pn pulse.Number) reference.Global {
	local := UniqueLocalRefWithPulse(pn)
	return reference.NewSelf(local)
}

// UniqueGlobalRefs generates multiple random unique reference.Global values.
func UniqueGlobalRefs(n int) []reference.Global {
	refs := make([]reference.Global, n)

	for i := 0; i < n; i++ {
		refs[i] = UniqueGlobalRef()
	}
	return refs
}
