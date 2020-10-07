// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package output

import (
	"github.com/insolar/assured-ledger/ledger-core/testutils/pulsewatcher/status"
)

const (
	TimeFormat = "15:04:05.999999"
)

type Printer interface {
	Print([]status.Node)
}
