// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
)

type One struct { // nolint
	foundation.BaseContract
}

var INSATTR_Check_API = true // nolint

func (t *One) Check() (bool, error) {
	return true, nil
}
