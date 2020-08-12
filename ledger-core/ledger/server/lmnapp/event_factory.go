// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/requests"
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
			CreateFn: func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return requests.NewSMRegisterRecordSet(ev)
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
