// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

func TestExample(t *testing.T) {
	inslogger.SetTestOutput(t, false)
	RunExample()
}
