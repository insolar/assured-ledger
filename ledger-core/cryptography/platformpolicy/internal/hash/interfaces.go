// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hash

import (
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
)

type AlgorithmProvider interface {
	Hash224bits() cryptography.Hasher
	Hash256bits() cryptography.Hasher
	Hash512bits() cryptography.Hasher
}
