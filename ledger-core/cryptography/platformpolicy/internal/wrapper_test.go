package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy/internal/hash"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy/internal/sign"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestEcdsaMarshalUnmarshal(t *testing.T) {
	data := gen.UniqueGlobalRef()

	kp := platformpolicy.NewKeyProcessor()
	provider := sign.NewECDSAProvider()

	hasher := hash.NewSHA3Provider().Hash512bits()

	privateKey, err := kp.GeneratePrivateKey()
	assert.NoError(t, err)

	signer := provider.DataSigner(privateKey, hasher)
	verifier := provider.DataVerifier(kp.ExtractPublicKey(privateKey), hasher)

	signature, err := signer.Sign(data.AsBytes())
	assert.NoError(t, err)

	assert.True(t, verifier.Verify(*signature, data.AsBytes()))
}
