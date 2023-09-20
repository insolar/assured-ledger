package output

import (
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher/status"
)

const (
	TimeFormat = "15:04:05.999999"
)

type Printer interface {
	Print([]status.Node)
}
