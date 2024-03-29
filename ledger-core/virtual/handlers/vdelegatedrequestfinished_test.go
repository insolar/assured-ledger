package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

func TestSMVDelegatedRequestFinished_FailIfCallNotRegistered(t *testing.T) {
	var (
		mc          = minimock.NewController(t)
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smObject    = object.NewStateMachineObject(smGlobalRef)
		smExecID    = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callMode    = isolation.CallTolerable
	)

	table := smObject.PendingTable.GetList(callMode)
	require.Equal(t, 0, table.Count())

	sm := SMVDelegatedRequestFinished{
		Payload: &rms.VDelegatedRequestFinished{
			Callee:    rms.NewReference(smExecID),
			CallFlags: rms.BuildCallFlags(callMode, isolation.CallDirty),
		},
	}

	execCtx := smachine.NewExecutionContextMock(mc)

	require.Panics(t, func() {
		sm.updateSharedState(execCtx, &smObject.SharedState)
	})

	mc.Finish()
}

func TestSMVDelegatedRequestFinished_TolerableUpdateSharedState_OneActiveCounter(t *testing.T) {
	var (
		mc          = minimock.NewController(t)
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smObject    = object.NewStateMachineObject(smGlobalRef)
		smExecID    = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callMode    = isolation.CallTolerable
	)

	smObject.PreviousExecutorOrderedPendingCount = 1

	table := smObject.PendingTable.GetList(callMode)
	table.Add(smExecID)
	require.Equal(t, 1, table.Count())
	require.Equal(t, 1, table.CountActive())

	sm := SMVDelegatedRequestFinished{
		Payload: &rms.VDelegatedRequestFinished{
			Callee:       rms.NewReference(smGlobalRef),
			CallOutgoing: rms.NewReference(smExecID),
			CallFlags:    rms.BuildCallFlags(callMode, isolation.CallDirty),
		},
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		CallBargeInWithParamMock.Return(true)

	sm.updateSharedState(execCtx, &smObject.SharedState)

	require.Equal(t, 1, table.CountFinish())
	require.Equal(t, 0, table.CountActive())
	require.Equal(t, uint8(1), smObject.PreviousExecutorOrderedPendingCount)
	mc.Finish()
}

func TestSMVDelegatedRequestFinished_TolerableUpdateSharedState_ManyActiveCounter(t *testing.T) {
	var (
		mc          = minimock.NewController(t)
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smObject    = object.NewStateMachineObject(smGlobalRef)
		smExecID    = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callMode    = isolation.CallTolerable
	)

	smObject.PreviousExecutorOrderedPendingCount = 2

	table := smObject.PendingTable.GetList(callMode)
	table.Add(smExecID)
	require.Equal(t, 1, table.Count())
	require.Equal(t, 1, table.CountActive())
	table.Add(gen.UniqueGlobalRefWithPulse(pd.PulseNumber))
	require.Equal(t, 2, table.Count())
	require.Equal(t, 2, table.CountActive())

	sm := SMVDelegatedRequestFinished{
		Payload: &rms.VDelegatedRequestFinished{
			Callee:       rms.NewReference(smGlobalRef),
			CallOutgoing: rms.NewReference(smExecID),
			CallFlags:    rms.BuildCallFlags(callMode, isolation.CallDirty),
		},
	}

	execCtx := smachine.NewExecutionContextMock(mc)

	sm.updateSharedState(execCtx, &smObject.SharedState)

	require.Equal(t, 1, table.CountFinish())
	require.Equal(t, 1, table.CountActive())
	require.Equal(t, uint8(2), smObject.PreviousExecutorOrderedPendingCount)
	mc.Finish()
}

func TestSMVDelegatedRequestFinished_IntolerableUpdateSharedStateUpdatePendingTable(t *testing.T) {
	var (
		mc          = minimock.NewController(t)
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smObject    = object.NewStateMachineObject(smGlobalRef)
		smExecID    = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callMode    = isolation.CallIntolerable
	)

	smObject.PreviousExecutorUnorderedPendingCount = 1

	table := smObject.PendingTable.GetList(callMode)
	table.Add(smExecID)
	require.Equal(t, 1, table.Count())
	require.Equal(t, 1, table.CountActive())

	sm := SMVDelegatedRequestFinished{
		Payload: &rms.VDelegatedRequestFinished{
			Callee:       rms.NewReference(smGlobalRef),
			CallOutgoing: rms.NewReference(smExecID),
			CallFlags:    rms.BuildCallFlags(callMode, isolation.CallDirty),
		},
	}
	execCtx := smachine.NewExecutionContextMock(mc)
	sm.updateSharedState(execCtx, &smObject.SharedState)

	require.Equal(t, 1, table.CountFinish())
	require.Equal(t, 0, table.CountActive())
	require.Equal(t, uint8(1), smObject.PreviousExecutorUnorderedPendingCount)
	mc.Finish()
}
