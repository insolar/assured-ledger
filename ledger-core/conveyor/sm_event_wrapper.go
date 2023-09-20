package conveyor

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type wrapEventSM struct {
	smachine.StateMachineDeclTemplate

	pn       pulse.Number
	ps       *PulseSlot
	createFn smachine.CreateFunc
}
