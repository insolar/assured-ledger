// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptography

import (
	"crypto"
	"hash"
)

type Hasher interface {
	hash.Hash

	Hash([]byte) []byte
}

type Signer interface {
	Sign([]byte) (*Signature, error)
}

type Verifier interface {
	Verify(Signature, []byte) bool
}

type PlatformCryptographyScheme interface {
	PublicKeySize() int
	SignatureSize() int

	ReferenceHashSize() int
	ReferenceHasher() Hasher

	IntegrityHashSize() int
	IntegrityHasher() Hasher

	DataSigner(crypto.PrivateKey, Hasher) Signer
	DigestSigner(key crypto.PrivateKey) Signer
	DataVerifier(crypto.PublicKey, Hasher) Verifier
	DigestVerifier(crypto.PublicKey) Verifier
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/cryptography.KeyProcessor -o ../testutils -s _mock.go -g
type KeyProcessor interface {
	GeneratePrivateKey() (crypto.PrivateKey, error)
	ExtractPublicKey(crypto.PrivateKey) crypto.PublicKey

	ExportPublicKeyPEM(crypto.PublicKey) ([]byte, error)
	ImportPublicKeyPEM([]byte) (crypto.PublicKey, error)

	ExportPrivateKeyPEM(crypto.PrivateKey) ([]byte, error)
	ImportPrivateKeyPEM([]byte) (crypto.PrivateKey, error)

	ExportPublicKeyBinary(crypto.PublicKey) ([]byte, error)
	ImportPublicKeyBinary([]byte) (crypto.PublicKey, error)
}
