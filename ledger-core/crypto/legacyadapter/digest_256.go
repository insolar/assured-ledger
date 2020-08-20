// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const SHA3Digest512as256 = cryptkit.DigestMethod("sha3-512-256")

type Sha3Digester256 struct {
	scheme cryptography.PlatformCryptographyScheme
}

func NewSha3Digester256(scheme cryptography.PlatformCryptographyScheme) Sha3Digester256 {
	return Sha3Digester256{
		scheme: scheme,
	}
}

func (pd Sha3Digester256) DigestData(reader io.Reader) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	_, err := io.Copy(hasher, reader)
	if err != nil {
		panic(err)
	}

	bytes := hasher.Sum(nil)
	bits := longbits.NewBits256FromBytes(bytes[:32])

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester256) DigestBytes(bytes []byte) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	bytes = hasher.Hash(bytes)
	bits := longbits.NewBits256FromBytes(bytes[:32])

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester256) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: pd, Hash: pd.scheme.IntegrityHasher()}
}

func (pd Sha3Digester256) GetDigestSize() int {
	return 32
}

func (pd Sha3Digester256) GetDigestMethod() cryptkit.DigestMethod {
	return SHA3Digest512as256
}

