// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestNewSha3Digester224(t *testing.T) {
	digester := NewSha3Digester224(scheme)

	require.Implements(t, (*cryptkit.DataDigester)(nil), digester)

	require.Equal(t, digester.scheme, scheme)
}

func TestSha3Digester224_GetDigestOf(t *testing.T) {
	digester := NewSha3Digester224(scheme)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digest := digester.DigestData(reader)
	require.Equal(t, 28, digest.FixedByteSize())

	expected := scheme.ReferenceHasher().Hash(b)

	require.Equal(t, expected, longbits.AsBytes(digest))

	digest = digester.DigestBytes(b)
	require.Equal(t, expected, longbits.AsBytes(digest))
}

func TestSha3Digester224_GetDigestMethod(t *testing.T) {
	digester := NewSha3Digester224(scheme)

	require.Equal(t, digester.GetDigestMethod(), SHA3Digest224)
}

