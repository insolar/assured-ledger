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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewEventFactory(ctx context.Context) *EventFactory {
	return &EventFactory{ ctx }
}

type EventFactory struct {
	ctx context.Context
}

func (p *EventFactory) InputEvent(ctx context.Context, event conveyor.InputEvent, context conveyor.InputContext) (conveyor.InputSetup, error) {
	panic("implement me")
}

func (p *EventFactory) PostMigrate(prevState conveyor.PulseSlotState, ps *conveyor.PulseSlot, m smachine.SlotMachineHolder) {
	switch {
	case ps.State() != conveyor.Present:
		return
	case prevState == conveyor.Present:
		panic(throw.IllegalState())
	}

	m.AddNewByFunc(p.ctx, datawriter.PlashCreate(), smachine.CreateDefaultValues{})
}
