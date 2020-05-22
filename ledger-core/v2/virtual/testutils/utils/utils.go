// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit"
)

func CmpStateFuncs(want, got interface{}) bool {
	return reflectkit.CodeOf(want) == reflectkit.CodeOf(got)
}
