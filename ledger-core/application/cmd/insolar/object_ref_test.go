package main

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

// ID and Reference serialization tests

func TestRecordID_String(t *testing.T) {
	id := gen.UniqueLocalRef()
	idStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(id.AsBytes()) + ".record"

	assert.Equal(t, idStr, reference.EncodeLocal(id))
}
func TestRecordRef_String(t *testing.T) {
	ref := gen.UniqueGlobalRef()
	expectedRefStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(ref.GetLocal().AsBytes())

	assert.Equal(t, expectedRefStr, reference.Encode(ref))
}
