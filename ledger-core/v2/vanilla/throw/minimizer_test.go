// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import "testing"

const testStackDebug = "" +
	"runtime/debug.Stack(0xc000313cc0, 0x1020601, 0x6)\n" +
	testStackClear

const testStackPanic = "" +
	"runtime/debug.Stack(0xc000313cc0, 0x1020601, 0x6)\n" +
	"\t/go1.14.2/src/runtime/debug/stack.go:24 +0xa4\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.RecoverSlotPanicWithStack(...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/api_panic.go:73\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.recoverSlotPanicAsUpdate(0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/state_update_type.go:66 +0x169\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*contextTemplate).discardAndUpdate(...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/context_basic.go:94\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*executionContext).executeNextStep.func1(0xc000336ab0, 0xc0002748c0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/context_exec.go:123 +0x126\n" +
	"panic(0xe6b060, 0x11875c0)\n" +
	"\t/go1.14.2/src/runtime/panic.go:975 +0x3f1\n" +
	testStackClear

const testStackValuable = "" +
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/execute.(*SMExecute).stepWaitObjectReady(0xc000352000, 0x11d51a0, 0xc000336ab0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/virtual/execute/execute.go:161 +0x7ce\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/execute.(*SMExecute).stepWaitObjectReady(0xc000352000, 0x11d51a0, 0xc000336ab0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/virtual/execute/execute.go:161 +0x7ce\n"

const testStackClear = "" +
	testStackValuable +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*executionContext).executeNextStep(0xc000336ab0, 0xc000336ab0, 0x1, 0x2208038, 0xc0003369c0, 0x101f001, 0x0, 0x0, 0x0, 0xb, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/context_exec.go:128 +0xfb\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine)._executeSlot.func1(0x11c8fa0, 0xc000336aa0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:222 +0x200\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker.(*SimpleSlotWorker).DetachableCall(0xc000336a80, 0xc000266b00, 0x1912640)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker/worker_simple.go:114 +0x49\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine)._executeSlot(0xc0001da800, 0xc000480000, 0x4, 0x11c4bc0, 0xc000336a80, 0x2711, 0x0, 0x0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:203 +0x210\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine).executeWorkingSlots(0xc0001da800, 0x5, 0x11c4bc0, 0xc000336a80)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:183 +0x1ba\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine).ScanOnce(0xc0001da800, 0x0, 0x11c4bc0, 0xc000336a80, 0xc0004473c8, 0x40dcbf, 0x20, 0xef77a0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:120 +0x462\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine).ScanNested.func1(0x11c4bc0, 0xc000336a80)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:35 +0x63\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker.(*AttachableSimpleSlotWorker).AttachAsNested(0xc00002ad90, 0xc0001da800, 0x11c8fa0, 0xc000336950, 0x2710, 0xc000336a50, 0x117e300)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker/worker_simple.go:44 +0x14b\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine).ScanNested(0xc0001da800, 0x11d51a0, 0xc000336a20, 0x271000274800, 0x11ac860, 0xc00002ad90, 0xfac100, 0xc000336960, 0x0, 0x40)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:34 +0x126\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor.(*PulseSlotMachine).stepPresentLoop(0xc000067900, 0x11d51a0, 0xc000336a20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sm_pulse_slot.go:207 +0x91\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*executionContext).executeNextStep(0xc000336a20, 0xc000336a20, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, ...)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/context_exec.go:128 +0xfb\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine)._executeSlot.func1(0x11c8fa0, 0xc000336950)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:222 +0x200\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker.(*SimpleSlotWorker).DetachableCall(0xc000336930, 0xc000266a80, 0x1912640)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker/worker_simple.go:114 +0x49\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine)._executeSlot(0xc0001da000, 0xc000400000, 0x2, 0x11c4bc0, 0xc000336930, 0x2711, 0x0, 0x0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:203 +0x210\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine).executeWorkingSlots(0xc0001da000, 0x6, 0x11c4bc0, 0xc000336930)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:183 +0x1ba\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine.(*SlotMachine).ScanOnce(0xc0001da000, 0xc00026ee00, 0x11c4bc0, 0xc000336930, 0xc000447e68, 0x40dcbf, 0x30, 0xf7c160)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/slot_machine_execute.go:120 +0x462\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor.(*PulseConveyor).runWorker.func2(0x11c4bc0, 0xc000336930)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/conveyor.go:488 +0x67\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker.(*AttachableSimpleSlotWorker).AttachTo(0xc00002ac70, 0xc0001da000, 0xc0001af500, 0xffffffff, 0xc00026ee20, 0x0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker/worker_simple.go:59 +0xda\n" +
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor.(*PulseConveyor).runWorker(0xc000072f00, 0x0, 0xc0001864e0, 0xc0001b46d0)\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/conveyor.go:487 +0x1ad\n" +
	"created by github.com/insolar/assured-ledger/ledger-core/v2/conveyor.(*PulseConveyor).StartWorker\n" +
	"\t/go/src/github.com/insolar/assured-ledger/ledger-core/v2/conveyor/conveyor.go:462 +0xe0\n"

func TestMinimizeText(t *testing.T) {
	println(
		string(MinimizeDebugStack([]byte(testStackDebug), "github.com/insolar/assured-ledger/ledger-core/v2/conveyor", true)))
}
