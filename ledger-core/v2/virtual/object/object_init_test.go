// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func TestInitViaCTMethod(t *testing.T) {
	var (
		// ctx = inslogger.TestContext(t)
		mc = minimock.NewController(t)

		pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{}) // TODO pulse data with changes
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
		smObject    = NewStateMachineObject(smGlobalRef, InitReasonCTMethod)
		// msgVStateReportCount = 0
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	defer mc.Finish()
	smObject.pulseSlot = &pulseSlot

	// TODO why i skip some steps and its still work?
	// TODO why if i use zero steps test throws panic but still pass?
	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepWaitState)
		stepChecker.AddStep(exec.stepGetObjectState) // TODO after first todo's there will be some checks
		stepChecker.AddStep(exec.stepWaitState)
		stepChecker.AddStep(exec.stepReadyToWork)
		stepChecker.AddStep(exec.stepWaitIndefinitely)
	}

	initCtx := smachine.NewInitializationContextMock(mc).
		ShareMock.Return(sharedStateData).
		PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
		JumpMock.Set(testutils.CheckWrapper(stepChecker, t))

	smObject.Init(initCtx)
	mc.Wait(10 * time.Second)
}
