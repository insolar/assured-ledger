package smachine_test

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type testsHelper struct {
	scanCountLimit int
	m              *smachine.SlotMachine
	wFactory       *sworker.AttachableSimpleSlotWorker
	neverSignal    *synckit.SignalVersion
}

func newTestsHelper() *testsHelper {
	res := testsHelper{}
	res.scanCountLimit = 1000

	signal := synckit.NewVersionedSignal()
	res.m = smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:    1000,
		PollingPeriod:   10 * time.Millisecond,
		PollingTruncate: 1 * time.Microsecond,
		ScanCountLimit:  res.scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	res.wFactory = sworker.NewAttachableSimpleSlotWorker()
	res.neverSignal = synckit.NewNeverSignal()

	return &res
}

func (h *testsHelper) add(ctx context.Context, sm smachine.StateMachine) {
	h.m.AddNew(ctx, sm, smachine.CreateDefaultValues{})
}

func (h *testsHelper) iter(fn func() bool) {
	for repeatNow := true; repeatNow && (fn == nil || fn()); {
		h.wFactory.AttachTo(h.m, h.neverSignal, uint32(h.scanCountLimit), func(worker smachine.AttachedSlotWorker) {
			repeatNow, _ = h.m.ScanOnce(0, worker)
		})
	}
}

func (h *testsHelper) migrate() {
	if !h.m.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}
}
