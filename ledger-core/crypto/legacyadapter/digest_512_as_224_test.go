package legacyadapter

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestNewSha3Digester512as224(t *testing.T) {
	digester := NewSha3Digester512as224(scheme)

	require.Implements(t, (*cryptkit.DataDigester)(nil), digester)

	require.Equal(t, digester.scheme, scheme)
}

func TestSha3Digester512as224_GetDigestOf(t *testing.T) {
	digester := NewSha3Digester512as224(scheme)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digest := digester.DigestData(reader)
	require.Equal(t, 28, digest.FixedByteSize())

	expected := scheme.IntegrityHasher().Hash(b)

	require.Equal(t, expected[:28], longbits.AsBytes(digest))

	digest = digester.DigestBytes(b)
	require.Equal(t, expected[:28], longbits.AsBytes(digest))
}

func TestSha3Digester512as224_GetDigestMethod(t *testing.T) {
	digester := NewSha3Digester512as224(scheme)

	require.Equal(t, digester.GetDigestMethod(), SHA3Digest512as224)
}

