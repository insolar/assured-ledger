// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalFromString(t *testing.T) {
	id := randLocal()
	idStr := "insolar:1" + base64.RawURLEncoding.EncodeToString(id.Bytes())
	id2, err := LocalFromString(idStr)
	require.NoError(t, err)

	assert.Equal(t, id, id2)
}
