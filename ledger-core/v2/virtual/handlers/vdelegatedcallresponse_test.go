// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func TestSMVDelegatedCallResponse_ErrorIfBargeInWasNotPublished(t *testing.T) {
	var (
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		bargeInID = gen.UniqueIDWithPulse(pd.PulseNumber)
		globalRef = reference.NewSelf(bargeInID)
	)

	sm := SMVDelegatedCallResponse{
		Payload: &payload.VDelegatedCallResponse{RefIn: globalRef},
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		GetPublishedGlobalAliasAndBargeInMock.Expect(globalRef).Return(smachine.SlotLink{}, smachine.BargeIn{}).
		ErrorMock.Expect(errors.New("bargeIn was not published")).Return(smachine.StateUpdate{})

	sm.stepProcess(execCtx)

	mc.Finish()
}

func TestSMVDelegatedCallResponse_ErrorIfCallBargeInFailed(t *testing.T) {
	var (
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		bargeInID = gen.UniqueIDWithPulse(pd.PulseNumber)
		globalRef = reference.NewSelf(bargeInID)
	)

	sm := SMVDelegatedCallResponse{
		Payload: &payload.VDelegatedCallResponse{RefIn: globalRef},
	}

	slotLink := smachine.DeadSlotLink()
	bargeIn := smachine.NewBargeInHolderMock(mc).CallWithParamMock.Return(false)
	execCtx := smachine.NewExecutionContextMock(mc).
		GetPublishedGlobalAliasAndBargeInMock.Expect(globalRef).Return(slotLink, bargeIn).
		ErrorMock.Expect(errors.New("fail to call BargeIn")).Return(smachine.StateUpdate{})

	sm.stepProcess(execCtx)

	mc.Finish()
}

func TestSMVDelegatedCallResponse_SuccessCallBargeIn(t *testing.T) {
	var (
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		bargeInID = gen.UniqueIDWithPulse(pd.PulseNumber)
		globalRef = reference.NewSelf(bargeInID)
	)

	sm := SMVDelegatedCallResponse{
		Payload: &payload.VDelegatedCallResponse{RefIn: globalRef},
	}

	slotLink := smachine.DeadSlotLink()
	bargeIn := smachine.NewBargeInHolderMock(mc).
		CallWithParamMock.Expect(sm.Payload).Return(true)
	execCtx := smachine.NewExecutionContextMock(mc).
		GetPublishedGlobalAliasAndBargeInMock.Expect(globalRef).Return(slotLink, bargeIn).
		StopMock.Return(smachine.StateUpdate{})

	sm.stepProcess(execCtx)

	mc.Finish()
}
