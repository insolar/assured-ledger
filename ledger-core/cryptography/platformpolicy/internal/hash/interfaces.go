package hash

import (
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
)

type AlgorithmProvider interface {
	Hash224bits() cryptography.Hasher
	Hash256bits() cryptography.Hasher
	Hash512bits() cryptography.Hasher
}
