// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hash

import (
	"golang.org/x/crypto/sha3"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type sha3Provider struct{}

func NewSHA3Provider() AlgorithmProvider {
	return &sha3Provider{}
}

func (*sha3Provider) Hash224bits() insolar.Hasher {
	return &hashWrapper{
		hash: sha3.New224(),
		sumFunc: func(b []byte) []byte {
			s := sha3.Sum224(b)
			return s[:]
		},
	}
}

func (*sha3Provider) Hash256bits() insolar.Hasher {
	return &hashWrapper{
		hash: sha3.New256(),
		sumFunc: func(b []byte) []byte {
			s := sha3.Sum256(b)
			return s[:]
		},
	}
}

func (*sha3Provider) Hash512bits() insolar.Hasher {
	return &hashWrapper{
		hash: sha3.New512(),
		sumFunc: func(b []byte) []byte {
			s := sha3.Sum512(b)
			return s[:]
		},
	}
}
