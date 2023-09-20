package cryptography

import (
	"crypto"
)

type Signature struct {
	raw []byte
}

func SignatureFromBytes(raw []byte) Signature {
	return Signature{raw: raw}
}

func (s *Signature) Bytes() []byte {
	return s.raw
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/cryptography.Service -s _mock.go -g
type Service interface {
	Signer
	GetPublicKey() (crypto.PublicKey, error)
	Verify(crypto.PublicKey, Signature, []byte) bool
}
