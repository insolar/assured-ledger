// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
)

// ID and Reference serialization tests

func TestRecordID_String(t *testing.T) {
	id := gen.ID()
	idStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(id.AsBytes()) + ".record"

	assert.Equal(t, idStr, id.String())
}
func TestRecordRef_String(t *testing.T) {
	ref := gen.Reference()
	expectedRefStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(ref.GetLocal().AsBytes())

	assert.Equal(t, expectedRefStr, ref.String())
}
