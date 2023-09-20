package legacyadapter

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestNewSha3Digester512(t *testing.T) {
	digester := NewSha3Digester512(scheme)

	require.Implements(t, (*cryptkit.DataDigester)(nil), digester)

	require.Equal(t, digester.scheme, scheme)
}

func TestSha3Digester512_GetDigestOf(t *testing.T) {
	digester := NewSha3Digester512(scheme)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digest := digester.DigestData(reader)
	require.Equal(t, 64, digest.FixedByteSize())
	require.Equal(t, scheme.IntegrityHashSize(), digest.FixedByteSize())

	expected := scheme.IntegrityHasher().Hash(b)

	require.Equal(t, expected, longbits.AsBytes(digest))

	digest = digester.DigestBytes(b)
	require.Equal(t, expected, longbits.AsBytes(digest))
}

func TestSha3Digester512_GetDigestMethod(t *testing.T) {
	digester := NewSha3Digester512(scheme)

	require.Equal(t, digester.GetDigestMethod(), SHA3Digest512)
}

