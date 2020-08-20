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

const SHA3Digest512 = cryptkit.DigestMethod("sha3-512")

func NewSha3Digester512(scheme cryptography.PlatformCryptographyScheme) Sha3Digester512 {
	return Sha3Digester512{
		scheme: scheme,
	}
}

type Sha3Digester512 struct {
	scheme cryptography.PlatformCryptographyScheme
}

func (pd Sha3Digester512) DigestPair(digest0 longbits.FoldableReader, digest1 longbits.FoldableReader) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	_, _ = digest0.WriteTo(hasher)
	_, _ = digest1.WriteTo(hasher)

	bytes := hasher.Sum(nil)
	bits := longbits.NewBits512FromBytes(bytes)

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester512) DigestData(reader io.Reader) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	_, err := io.Copy(hasher, reader)
	if err != nil {
		panic(err)
	}

	bytes := hasher.Sum(nil)
	bits := longbits.NewBits512FromBytes(bytes)

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester512) DigestBytes(bytes []byte) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	bytes = hasher.Hash(bytes)
	bits := longbits.NewBits512FromBytes(bytes)

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester512) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: pd, Hash: pd.scheme.IntegrityHasher()}
}

func (pd Sha3Digester512) GetDigestSize() int {
	return 64
}

func (pd Sha3Digester512) GetDigestMethod() cryptkit.DigestMethod {
	return SHA3Digest512
}

