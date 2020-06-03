// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package args

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit"

func IsNil(v interface{}) bool {
	return reflectkit.IsNil(v)
}

type ShuffleFunc func(n int, swap func(i, j int))
