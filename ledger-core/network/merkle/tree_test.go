// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkle

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
)

func TestFromList(t *testing.T) {
	cs := platformpolicy.NewPlatformCryptographyScheme()

	mt, err := treeFromHashList([][]byte{
		cs.IntegrityHasher().Hash([]byte("123")),
		cs.IntegrityHasher().Hash([]byte("456")),
	}, cs.IntegrityHasher())
	require.NoError(t, err)

	root := mt.Root()

	require.NotNil(t, root)
}
