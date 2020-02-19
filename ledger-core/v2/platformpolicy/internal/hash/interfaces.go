// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hash

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type AlgorithmProvider interface {
	Hash224bits() insolar.Hasher
	Hash256bits() insolar.Hasher
	Hash512bits() insolar.Hasher
}
