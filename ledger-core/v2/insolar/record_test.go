// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
)

// ID and Reference serialization tests

func TestNewIDFromBytes(t *testing.T) {
	id := gen.ID()
	actualID := insolar.NewIDFromBytes(id.AsBytes())
	require.Equal(t, id, actualID)

	insolar.NewIDFromBytes(nil)
}

func TestNewIDFromString(t *testing.T) {
	id := gen.ID()
	idStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(id.AsBytes())
	id2, err := insolar.NewIDFromString(idStr)
	require.NoError(t, err)

	assert.Equal(t, id, id2)
}

func TestRecordID_String(t *testing.T) {
	id := gen.ID()
	idStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(id.AsBytes()) + ".record"

	assert.Equal(t, idStr, id.String())
}

func TestNewRefFromString(t *testing.T) {
	recordID := gen.ID()
	domainID := gen.ID()
	refStr := "insolar:1" +
		base64.RawURLEncoding.EncodeToString(recordID.AsBytes()) +
		insolar.RecordRefIDSeparator + "1" +
		base64.RawURLEncoding.EncodeToString(domainID.AsBytes())

	expectedRef := insolar.NewGlobalReference(recordID, domainID)
	actualRef, err := insolar.NewReferenceFromString(refStr)
	require.NoError(t, err)

	assert.Equal(t, expectedRef, actualRef)
}

func TestRecordRef_String(t *testing.T) {
	ref := gen.Reference()
	expectedRefStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(ref.GetLocal().AsBytes())

	assert.Equal(t, expectedRefStr, ref.String())
}

func TestNewReferenceFromString(t *testing.T) {
	origin := gen.Reference()
	decoded, err := insolar.NewReferenceFromString(origin.String())
	require.NoError(t, err)
	assert.Equal(t, origin, decoded)
}
