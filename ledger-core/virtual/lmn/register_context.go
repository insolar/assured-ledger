// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmn

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type IncomingRegistrationStatus int8

const (
	RegistrationNot IncomingRegistrationStatus = iota
	RegistrationPrepared
	RegistrationOk
	RegistrationFail
)

type RegistrationCtx struct {
	prevBranchRef reference.Global
	branchRef     reference.Global

	prevTrunkRef reference.Global
	trunkRef     reference.Global

	incomingRegistrationStatus IncomingRegistrationStatus

	objectRef                  reference.Global
	safeResponseCounter        SafeResponseCounter
	lmnSafeResponseCounterLink smachine.SharedDataLink
}

func (c *RegistrationCtx) Rollback() {
	if c.prevBranchRef.IsEmpty() {
		panic(throw.IllegalState())
	}
	if c.prevTrunkRef.IsEmpty() {
		panic(throw.IllegalState())
	}

	c.branchRef = c.prevBranchRef
	c.trunkRef = c.prevTrunkRef
}

func (c *RegistrationCtx) SetNewReferences(trunk reference.Global, branch reference.Global) {
	c.prevTrunkRef = c.trunkRef
	c.prevBranchRef = c.branchRef

	c.trunkRef = trunk
	c.branchRef = branch
}

func (c *RegistrationCtx) Init(object reference.Global, trunk reference.Global) {
	c.objectRef = object
	c.trunkRef = trunk
}

func (c *RegistrationCtx) IncrementSafeResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return CounterIncrement(ctx, c.lmnSafeResponseCounterLink, 1)
}

func (c *RegistrationCtx) DecrementSafeResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return CounterDecrement(ctx, c.lmnSafeResponseCounterLink)
}

func (c *RegistrationCtx) AwaitZeroSafeResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return CounterAwaitZero(ctx, c.lmnSafeResponseCounterLink)
}

func (c *RegistrationCtx) GetObjectReference() reference.Global {
	if c.objectRef.IsEmpty() {
		panic(throw.FailHere("should be called only after Lifeline creation"))
	}
	return c.objectRef
}

func (c *RegistrationCtx) BranchRef() reference.Global {
	return c.branchRef
}

func (c *RegistrationCtx) TrunkRef() reference.Global {
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
	return &RegistrationCtx{trunkRef: trunk}
}
