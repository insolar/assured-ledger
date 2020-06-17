// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

func TestExample(t *testing.T) {
	inslogger.SetTestOutput(t, false)
	RunExample(5)
	/*
	RunExample(1)
	RunExample(3)
	RunExample(8)
	RunExample(0)
	RunExample(-1)
	 */
}
