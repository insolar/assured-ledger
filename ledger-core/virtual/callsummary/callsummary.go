// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package callsummary

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tables"
)

type pulseNumber = pulse.Number

type SummarySyncKey struct {
	objectRef reference.Global
}

func BuildSummarySyncKey(objectRef reference.Global) SummarySyncKey {
	return SummarySyncKey{objectRef: objectRef}
}

type SummarySharedKey struct {
	pulseNumber
}

func BuildSummarySharedKey(pulse pulse.Number) SummarySharedKey {
	return SummarySharedKey{pulseNumber: pulse}
}

func NewStateMachineCallSummary(pulse pulse.Number) *SMCallSummary {
	return &SMCallSummary{
		pulse: pulse,
	}
}

type SMCallSummary struct {
	smachine.StateMachineDeclTemplate

	pulse  pulse.Number
	shared SharedCallSummary
}

type SharedCallSummary struct {
	Requests tables.ObjectsRequestsTable
}

func (sm *SMCallSummary) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (sm *SMCallSummary) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

func (sm *SMCallSummary) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMCallSummary) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	sm.shared = SharedCallSummary{Requests: tables.NewObjectRequestTable()}

	sdl := ctx.Share(&sm.shared, 0)
	if !ctx.Publish(SummarySharedKey{pulseNumber: sm.pulse}, sdl) {
		return ctx.Stop()
	}

	ctx.SetDefaultMigration(sm.stepMigrate)

	return ctx.Jump(sm.stepLoop)
}

func (sm *SMCallSummary) stepMigrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (sm *SMCallSummary) stepLoop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

type SharedStateAccessor struct {
	smachine.SharedDataLink
}

func (v SharedStateAccessor) Prepare(fn func(*SharedCallSummary)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedCallSummary))
		return false
	})
}

func GetSummarySMSharedAccessor(
	ctx smachine.ExecutionContext,
	pulse pulse.Number,
) (SharedStateAccessor, bool) {
	if v := ctx.GetPublishedLink(BuildSummarySharedKey(pulse)); v.IsAssignableTo((*SharedCallSummary)(nil)) {
		return SharedStateAccessor{v}, true
	}
	return SharedStateAccessor{}, false
}

type SyncAccessor struct {
	smachine.SharedDataLink
}

func (v SyncAccessor) Prepare(fn func(*smachine.SyncLink)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*smachine.SyncLink))
		return false
	})
}

func GetSummarySMSyncAccessor(
	ctx smachine.ExecutionContext,
	objectRef reference.Global,
) (SyncAccessor, bool) {
	if v := ctx.GetPublishedLink(BuildSummarySyncKey(objectRef)); v.IsAssignableTo((*smachine.SyncLink)(nil)) {
		return SyncAccessor{v}, true
	}
	return SyncAccessor{}, false
}
