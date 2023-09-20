package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
)

func TestSMVDelegatedCallResponse_ErrorIfBargeInWasNotPublished(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		globalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
	)

	sm := SMVDelegatedCallResponse{
		Payload: &rms.VDelegatedCallResponse{
			ResponseDelegationSpec: rms.CallDelegationToken{
				Outgoing: rms.NewReference(globalRef),
			},
		},
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		GetPublishedGlobalAliasAndBargeInMock.Expect(execute.DelegationTokenAwaitKey{Outgoing: globalRef}).Return(smachine.SlotLink{}, smachine.BargeIn{}).
		ErrorMock.Expect(errors.New("bargeIn was not published")).Return(smachine.StateUpdate{})

	sm.stepProcess(execCtx)

	mc.Finish()
}

func TestSMVDelegatedCallResponse_ErrorIfCallBargeInFailed(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		globalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
	)

	sm := SMVDelegatedCallResponse{
		Payload: &rms.VDelegatedCallResponse{
			ResponseDelegationSpec: rms.CallDelegationToken{
				Outgoing: rms.NewReference(globalRef),
			},
		},
	}

	slotLink := smachine.DeadSlotLink()
	bargeIn := smachine.NewBargeInHolderMock(mc).CallWithParamMock.Return(false)
	execCtx := smachine.NewExecutionContextMock(mc).
		GetPublishedGlobalAliasAndBargeInMock.Expect(execute.DelegationTokenAwaitKey{Outgoing: globalRef}).Return(slotLink, bargeIn).
		ErrorMock.Expect(errors.New("fail to call BargeIn")).Return(smachine.StateUpdate{})

	sm.stepProcess(execCtx)

	mc.Finish()
}

func TestSMVDelegatedCallResponse_SuccessCallBargeIn(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		globalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
	)

	sm := SMVDelegatedCallResponse{
		Payload: &rms.VDelegatedCallResponse{
			ResponseDelegationSpec: rms.CallDelegationToken{
				Outgoing: rms.NewReference(globalRef),
			},
		},
	}

	slotLink := smachine.DeadSlotLink()
	bargeIn := smachine.NewBargeInHolderMock(mc).
		CallWithParamMock.Expect(sm.Payload).Return(true)
	execCtx := smachine.NewExecutionContextMock(mc).
		GetPublishedGlobalAliasAndBargeInMock.Expect(execute.DelegationTokenAwaitKey{Outgoing: globalRef}).Return(slotLink, bargeIn).
		StopMock.Return(smachine.StateUpdate{})

	sm.stepProcess(execCtx)

	mc.Finish()
}
