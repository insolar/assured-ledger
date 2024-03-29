package lmnapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/requests"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewEventFactory(ctx context.Context) *EventFactory {
	return &EventFactory{ ctx }
}

type EventFactory struct {
	ctx context.Context
}

func (p *EventFactory) InputEvent(_ context.Context, event conveyor.InputEvent, _ conveyor.InputContext) (conveyor.InputSetup, error) {
	switch ev := event.(type) {
	case inspectsvc.RegisterRequestSet:
		return conveyor.InputSetup{
			CreateFn: func(smachine.ConstructionContext) smachine.StateMachine {
				return requests.NewSMRegisterRecordSet(ev)
			},
		}, nil
	case inspectsvc.VerifyRequestSet:
		return conveyor.InputSetup{
			CreateFn: func(smachine.ConstructionContext) smachine.StateMachine {
				return requests.NewSMVerifyRecordSet(inspectsvc.RegisterRequestSet(ev))
			},
		}, nil
	case *rms.LReadRequest:
		return conveyor.InputSetup{
			CreateFn: func(smachine.ConstructionContext) smachine.StateMachine {
				return requests.NewSMRead(ev)
			},
		}, nil
	default:
		panic(throw.Unsupported())
	}
}

func (p *EventFactory) PostMigrate(prevState conveyor.PulseSlotState, ps *conveyor.PulseSlot, m smachine.SlotMachineHolder) {
	switch {
	case ps.State() != conveyor.Present:
		return
	case prevState == conveyor.Present:
		panic(throw.Impossible())
	}

	m.AddNewByFunc(p.ctx, datawriter.PlashCreate(), smachine.CreateDefaultValues{})
}
