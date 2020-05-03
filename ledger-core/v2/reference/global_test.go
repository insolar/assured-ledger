// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/base64"
	"encoding/binary"
	"sync/atomic"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

var uniqueSeq uint32

func getUnique() uint32 {
	return atomic.AddUint32(&uniqueSeq, 1)
}

func randLocal() Local {
	var id Local

	f := fuzz.New().NilChance(0).Funcs(func(id *Local, c fuzz.Continue) {
		var hash [LocalBinaryHashSize]byte
		c.Fuzz(&hash)
		binary.BigEndian.PutUint32(hash[LocalBinaryHashSize-4:], getUnique())

		pn := pulse.OfTime(time.Now())

		*id = NewLocal(pn, 0, BytesToLocalHash(hash[:]))
	})
	f.Fuzz(&id)

	return id
}

func TestFromString(t *testing.T) {
	recordID := randLocal()
	domainID := randLocal()
	refStr := "insolar:1" +
		base64.RawURLEncoding.EncodeToString(recordID.AsBytes()) +
		string(RecordRefIDSeparator) + "1" +
		base64.RawURLEncoding.EncodeToString(domainID.AsBytes())

	expectedRef := New(domainID, recordID)
	actualRef, err := GlobalFromString(refStr)
	require.NoError(t, err)

	assert.Equal(t, expectedRef, actualRef)
}
