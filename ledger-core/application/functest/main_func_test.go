// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"os"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
)

func TestMain(m *testing.M) {
	os.Exit(launchnet.Run(func() int {
		return m.Run()
	}))
}
