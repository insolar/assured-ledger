// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

func NewEventFactory() *EventFactory {
	return &EventFactory{}
}

type EventFactory struct {

}

func (p *EventFactory) InputEvent(ctx context.Context, event conveyor.InputEvent, context conveyor.InputContext) (conveyor.InputSetup, error) {
	panic("implement me")
}

func (p *EventFactory) SetupComponents(context.Context, injector.DependencyInjector, managed.RegisterComponentFunc) {}

func (p *EventFactory) PostMigrate(ps *conveyor.PulseSlot, m smachine.SlotMachineHolder) {

}
