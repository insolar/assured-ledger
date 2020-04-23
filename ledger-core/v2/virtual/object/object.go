// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type Info struct {
	Reference     insolar.Reference
	descriptor    descriptor.ObjectDescriptor
	Deactivated   bool
	IsReadyToWork bool

	ImmutableExecute smachine.SyncLink
	MutableExecute   smachine.SyncLink
	ReadyToWork      smachine.SyncLink

	PreviousExecutorState payload.PreviousExecutorState
}

func (i *Info) SetDescriptor(prototype *insolar.Reference, memory []byte) {
	i.descriptor = descriptor.NewObjectDescriptor(
		i.Reference, insolar.ID{}, prototype, memory, insolar.Reference{}, nil,
	)
}

func (i *Info) Deactivate() {
	i.Deactivated = true
}

func (i *Info) Descriptor() descriptor.ObjectDescriptor {
	return i.descriptor
}

type SharedState struct {
	Info
}

func NewStateMachineObject(objectReference insolar.Reference, exists bool) *SMObject {
	return &SMObject{
		SharedState: SharedState{
			Info: Info{Reference: objectReference},
		},
		oldObject: exists,
	}
}

type SMObject struct {
	smachine.StateMachineDeclTemplate

	SharedState

	readyToWorkCtl smsync.BoolConditionalLink

	oldObject bool
}

/* -------- Declaration ------------- */

func (sm *SMObject) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
}

func (sm *SMObject) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

/* -------- Instance ------------- */

func (sm *SMObject) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMObject) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	sm.readyToWorkCtl = smsync.NewConditionalBool(false, "readyToWork")
	sm.ReadyToWork = sm.readyToWorkCtl.SyncLink()

	sm.ImmutableExecute = smsync.NewSemaphore(30, "immutable calls").SyncLink()
	sm.MutableExecute = smsync.NewSemaphore(1, "mutable calls").SyncLink() // TODO here we need an ORDERED queue

	sdl := ctx.Share(&sm.SharedState, 0)
	if !ctx.Publish(sm.Reference.String(), sdl) {
		return ctx.Stop()
	}
	return ctx.Jump(sm.stepReadyToWork)
}

func (sm *SMObject) stepReadyToWork(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.ApplyAdjustment(sm.readyToWorkCtl.NewValue(true))

	return ctx.Jump(sm.stepWaitIndefinitely)
}

func (sm *SMObject) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
