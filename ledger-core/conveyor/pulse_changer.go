package conveyor

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
)

type PulseChanger interface {
	PreparePulseChange(out PreparePulseCallbackFunc)
	CancelPulseChange()
	CommitPulseChange()
}

var stubChanger PulseChanger = pulseChanger{}

type pulseChanger struct {}

func (pulseChanger) PreparePulseChange(outFn PreparePulseCallbackFunc) {
	if outFn != nil {
		// TODO temporary hack
		outFn(beat.AckData{})
	}
}

func (p pulseChanger) CancelPulseChange() {}
func (p pulseChanger) CommitPulseChange() {}


