package throw

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testStackDebugCall = "runtime/debug.Stack(0xc000313cc0, 0x1020601, 0x6)\n" +
	"\t/go1.14.2/src/runtime/debug/stack.go:24 +0xa4\n"

const testStackDebug = testStackDebugCall +
	testStackClear

const testStackPanic = testStackDebugCall +
	testStackPanicNoDebug

const testStackPanicNoDebug = testStackPanicDefer +
	testPanic +
	testStackClear

const testStackPanicDefer = "github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.RecoverSlotPanicWithStack(...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/api_panic.go:73\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.recoverSlotPanicAsUpdate(0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/state_update_type.go:66 +0x169\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*contextTemplate).discardAndUpdate(...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/context_basic.go:94\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*executionContext).executeNextStep.func1(0xc000336ab0, 0xc0002748c0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/context_exec.go:123 +0x126\n"

const testPanic = "panic(0xe6b060, 0x11875c0)\n" +
	"\t/go1.14.2/src/runtime/panic.go:975 +0x3f1\n"

const testStackValuable = testStackValuableOnce + testStackValuableOnce

const testStackValuableOnce = "github.com/insolar/assured-ledger/ledger-core/virtual/execute.(*SMExecute).stepWaitObjectReady(0xc000352000, 0x11d51a0, 0xc000336ab0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/virtual/execute/execute.go:161 +0x7ce\n"

const testStackBoundary = "github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*executionContext).executeNextStep(0xc000336ab0, 0xc000336ab0, 0x1, 0x2208038, 0xc0003369c0, 0x101f001, 0x0, 0x0, 0x0, 0xb, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/context_exec.go:128 +0xfb\n"

const testStackClear = testStackValuable +
	testStackBoundary +
	testStackUseless

const testStackUseless = "github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine)._executeSlot.func1(0x11c8fa0, 0xc000336aa0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:222 +0x200\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker.(*SimpleSlotWorker).DetachableCall(0xc000336a80, 0xc000266b00, 0x1912640)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/sworker/worker_simple.go:114 +0x49\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine)._executeSlot(0xc0001da800, 0xc000480000, 0x4, 0x11c4bc0, 0xc000336a80, 0x2711, 0x0, 0x0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:203 +0x210\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine).executeWorkingSlots(0xc0001da800, 0x5, 0x11c4bc0, 0xc000336a80)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:183 +0x1ba\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine).ScanOnce(0xc0001da800, 0x0, 0x11c4bc0, 0xc000336a80, 0xc0004473c8, 0x40dcbf, 0x20, 0xef77a0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:120 +0x462\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine).ScanNested.func1(0x11c4bc0, 0xc000336a80)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:35 +0x63\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker.(*AttachableSimpleSlotWorker).AttachAsNested(0xc00002ad90, 0xc0001da800, 0x11c8fa0, 0xc000336950, 0x2710, 0xc000336a50, 0x117e300)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/sworker/worker_simple.go:44 +0x14b\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine).ScanNested(0xc0001da800, 0x11d51a0, 0xc000336a20, 0x271000274800, 0x11ac860, 0xc00002ad90, 0xfac100, 0xc000336960, 0x0, 0x40)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:34 +0x126\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor.(*PulseSlotMachine).stepPresentLoop(0xc000067900, 0x11d51a0, 0xc000336a20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/sm_pulse_slot.go:207 +0x91\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*executionContext).executeNextStep(0xc000336a20, 0xc000336a20, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/context_exec.go:128 +0xfb\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine)._executeSlot.func1(0x11c8fa0, 0xc000336950)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:222 +0x200\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker.(*SimpleSlotWorker).DetachableCall(0xc000336930, 0xc000266a80, 0x1912640)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/sworker/worker_simple.go:114 +0x49\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine)._executeSlot(0xc0001da000, 0xc000400000, 0x2, 0x11c4bc0, 0xc000336930, 0x2711, 0x0, 0x0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:203 +0x210\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine).executeWorkingSlots(0xc0001da000, 0x6, 0x11c4bc0, 0xc000336930)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:183 +0x1ba\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.(*SlotMachine).ScanOnce(0xc0001da000, 0xc00026ee00, 0x11c4bc0, 0xc000336930, 0xc000447e68, 0x40dcbf, 0x30, 0xf7c160)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/slot_machine_execute.go:120 +0x462\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor.(*PulseConveyor).runWorker.func2(0x11c4bc0, 0xc000336930)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/conveyor.go:488 +0x67\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker.(*AttachableSimpleSlotWorker).AttachTo(0xc00002ac70, 0xc0001da000, 0xc0001af500, 0xffffffff, 0xc00026ee20, 0x0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/sworker/worker_simple.go:59 +0xda\n" +
	"github.com/insolar/assured-ledger/ledger-core/conveyor.(*PulseConveyor).runWorker(0xc000072f00, 0x0, 0xc0001864e0, 0xc0001b46d0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/conveyor.go:487 +0x1ad\n" +
	testCreatedBy

const testCreatedBy = "created by github.com/insolar/assured-ledger/ledger-core/conveyor.(*PulseConveyor).StartWorker\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/conveyor/conveyor.go:462 +0xe0\n"

const testBoundary = "github.com/insolar/assured-ledger/ledger-core/conveyor"

func TestMinimizeDebugStack(t *testing.T) {
	require.Equal(t, []byte(testStackValuable), MinimizeDebugStack([]byte(testStackDebug), testBoundary, false))
	require.Equal(t, []byte(testStackValuable+testStackBoundary), MinimizeDebugStack([]byte(testStackDebug), testBoundary, true))

	require.Equal(t, []byte(testStackClear), MinimizeDebugStack([]byte(testStackDebug), "not found", false))
	require.Equal(t, []byte(testStackClear), MinimizeDebugStack([]byte(testStackDebug), "not found", true))

	require.Equal(t, []byte(testStackUseless), MinimizeDebugStack([]byte(testStackUseless), testBoundary, false))
	require.Equal(t, []byte(testStackUseless), MinimizeDebugStack([]byte(testStackUseless), testBoundary, true))

	require.Equal(t, []byte(testStackUseless), MinimizeDebugStack([]byte(testStackDebugCall+testStackUseless), testBoundary, false))
	require.Equal(t, []byte(testStackUseless), MinimizeDebugStack([]byte(testStackDebugCall+testStackUseless), testBoundary, true))
}

func TestMinimizePanicStack(t *testing.T) {
	const testPanicValuable = testPanic + testStackValuable

	require.Equal(t, []byte(testPanicValuable), MinimizePanicStack([]byte(testStackPanic), testBoundary, false))
	require.Equal(t, []byte(testPanicValuable+testStackBoundary), MinimizePanicStack([]byte(testStackPanic), testBoundary, true))

	require.Equal(t, []byte(testStackPanicNoDebug), MinimizePanicStack([]byte(testStackPanic), "not found", false))
	require.Equal(t, []byte(testStackPanicNoDebug), MinimizePanicStack([]byte(testStackPanic), "not found", true))

	const testStackPanicUseless = testStackPanicDefer + testPanic + testStackUseless

	require.Equal(t, []byte(testStackPanicUseless), MinimizePanicStack([]byte(testStackPanicUseless), testBoundary, false))
	require.Equal(t, []byte(testStackPanicUseless), MinimizePanicStack([]byte(testStackPanicUseless), testBoundary, true))
}

func TestMinimizeMismatchedStack(t *testing.T) {
	require.Equal(t, []byte(testStackValuable), MinimizePanicStack([]byte(testStackDebug), testBoundary, false))
	require.Equal(t, []byte(testStackValuable+testStackBoundary), MinimizePanicStack([]byte(testStackDebug), testBoundary, true))

	require.Equal(t, []byte(testStackClear), MinimizePanicStack([]byte(testStackDebug), "not found", false))
	require.Equal(t, []byte(testStackClear), MinimizePanicStack([]byte(testStackDebug), "not found", true))

	require.Equal(t, []byte(testStackUseless), MinimizePanicStack([]byte(testStackUseless), testBoundary, false))
	require.Equal(t, []byte(testStackUseless), MinimizePanicStack([]byte(testStackUseless), testBoundary, true))

	require.Equal(t, []byte(testStackUseless), MinimizePanicStack([]byte(testStackDebugCall+testStackUseless), testBoundary, false))
	require.Equal(t, []byte(testStackUseless), MinimizePanicStack([]byte(testStackDebugCall+testStackUseless), testBoundary, true))
}

func TestMinimizeStackTrace(t *testing.T) {
	require.Nil(t, MinimizeStackTrace(nil, testBoundary, false))
	require.Equal(t, testStackValuable,
		MinimizeStackTrace(stackTrace{data: []byte(testStackDebug)}, testBoundary, false).StackTraceAsText())

	st := StackTrace(stackTrace{data: []byte(testStackUseless)})
	require.Equal(t, st, MinimizeStackTrace(st, testBoundary, false))
}
