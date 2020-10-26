// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package vnlmn

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type rawValue struct {
	val reference.Global
}

func (r rawValue) GetReference() reference.Global {
	return r.val
}

func (r rawValue) TryPullReference() reference.Global {
	return r.val
}

func NewRawValue(global reference.Global) rmsbox.ReferenceProvider {
	return &rawValue{val: global}
}

type IncomingRegistrationStatus int8

const (
	RegistrationNot IncomingRegistrationStatus = iota
	RegistrationPrepared
	RegistrationOk
	RegistrationFail
)

type RegistrationCtx struct {
	prevBranchRef rmsbox.ReferenceProvider
	branchRef     rmsbox.ReferenceProvider

	prevTrunkRef rmsbox.ReferenceProvider
	trunkRef     rmsbox.ReferenceProvider

	incomingRegistrationStatus IncomingRegistrationStatus

	objectRef                  rmsbox.ReferenceProvider
	safeResponseCounter        SafeResponseCounter
	lmnSafeResponseCounterLink smachine.SharedDataLink
}

func (c *RegistrationCtx) Rollback() {
	if c.prevBranchRef == nil {
		panic(throw.IllegalState())
	}
	if c.prevTrunkRef == nil {
		panic(throw.IllegalState())
	}

	c.branchRef = c.prevBranchRef
	c.trunkRef = c.prevTrunkRef
}

func (c *RegistrationCtx) SetNewReferences(trunk rmsbox.ReferenceProvider, branch rmsbox.ReferenceProvider) {
	c.prevTrunkRef = c.trunkRef
	c.prevBranchRef = c.branchRef

	c.trunkRef = trunk
	c.branchRef = branch
}

func (c *RegistrationCtx) Init(object reference.Global, trunk reference.Global) {
	c.objectRef = NewRawValue(object)
	c.trunkRef = NewRawValue(trunk)
}

func (c *RegistrationCtx) IncrementSafeResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return CounterIncrement(ctx, c.lmnSafeResponseCounterLink, 1)
}

func (c *RegistrationCtx) DecrementSafeResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return CounterDecrement(ctx, c.lmnSafeResponseCounterLink)
}

func (c *RegistrationCtx) WaitForAllSafeResponses(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return CounterAwaitZero(ctx, c.lmnSafeResponseCounterLink)
}

func (c *RegistrationCtx) BranchRef() rmsbox.ReferenceProvider {
	return c.branchRef
}

func (c *RegistrationCtx) TrunkRef() rmsbox.ReferenceProvider {
	return c.trunkRef
}

func (c *RegistrationCtx) IncomingRegistered() bool {
	return c.incomingRegistrationStatus == RegistrationOk
}

func (c *RegistrationCtx) IncomingRegistrationInProgress() bool {
	return c.incomingRegistrationStatus == RegistrationPrepared
}

func (c *RegistrationCtx) IncomingRegisterNext() {
	switch c.incomingRegistrationStatus {
	case RegistrationNot:
		c.incomingRegistrationStatus = RegistrationPrepared
	case RegistrationPrepared:
		c.incomingRegistrationStatus = RegistrationOk
	default:
		panic(throw.WithDetails(throw.IllegalState(), struct {
			status IncomingRegistrationStatus
		}{c.incomingRegistrationStatus}))
	}
}

func (c *RegistrationCtx) IncomingRegisterRollback() {
	if c.incomingRegistrationStatus != RegistrationPrepared {
		panic(throw.WithDetails(throw.IllegalState(), struct {
			status IncomingRegistrationStatus
		}{c.incomingRegistrationStatus}))
	}

	c.incomingRegistrationStatus = RegistrationFail
}

func (c *RegistrationCtx) IncomingRegisterOverride() {
	if c.incomingRegistrationStatus != RegistrationFail {
		panic(throw.WithDetails(throw.IllegalState(), struct {
			status IncomingRegistrationStatus
		}{c.incomingRegistrationStatus}))
	}

	c.incomingRegistrationStatus = RegistrationOk
}

func NewRegistrationCtx(ctx smachine.InitializationContext) *RegistrationCtx {
	rCtx := &RegistrationCtx{}
	rCtx.lmnSafeResponseCounterLink = ctx.Share(&rCtx.safeResponseCounter, 0)
	return rCtx
}

func NewDummyRegistrationCtx(trunk reference.Global) *RegistrationCtx {
	return &RegistrationCtx{trunkRef: NewRawValue(trunk)}
}
