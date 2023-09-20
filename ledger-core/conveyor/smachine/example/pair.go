package example

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SharedPairData struct {
	players     [2]smachine.BargeIn
	gameFactory GameFactoryFunc
}

type SharedPairDataLink struct {
	sharedData smachine.SharedDataLink
}

func (v SharedPairDataLink) PrepareAccess(fn func(*SharedPairData) (wakeup bool)) smachine.SharedDataAccessor {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return v.sharedData.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*SharedPairData))
	})
}
