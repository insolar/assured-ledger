package example

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

func RunExample(machineLogger smachine.SlotMachineLogger) {
	const scanCountLimit = 1e4

	/*** Initialize SlotMachine ***/
	signal := synckit.NewVersionedSignal()
	sm := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:         100,
		PollingPeriod:        100 * time.Millisecond,
		PollingTruncate:      1 * time.Millisecond,
		BoostNewSlotDuration: 0,
		ScanCountLimit:       scanCountLimit,
		LogAdapterCalls:      true,
		SlotMachineLogger:    machineLogger,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	/*** Add injectables ***/

	sm.AddDependency(NewGameAdapter(context.Background(), NewGameChooseService()))

	/*** Add SMs ***/

	const numberOfPlayers = 5
	const numberOfRooms = 1

	PlayRoomLimiter = smsync.NewFixedSemaphore(numberOfRooms, "rooms")

	for i := 0; i < numberOfPlayers; i++ {
		sm.AddNew(context.Background(), &PlayerSM{}, smachine.CreateDefaultValues{})
	}

	/*** Run the SlotMachine (and all SMs) ***/

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	for {
		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		wakeupSignal := signal.Mark()
		workerFactory.AttachTo(sm, neverSignal, scanCountLimit, func(worker smachine.AttachedSlotWorker) {
			repeatNow, nextPollTime = sm.ScanOnce(0, worker)
		})
		switch {
		case repeatNow:
			// there are active SMs, so we can start next cycle immediately
			continue
		case !nextPollTime.IsZero():
			// there are only sleeping SMs or waiting for the specific time
			//
			// here it do it simplistically as we also have to look for the (wakeupSignal)
			// as it can be triggered by async result from adapters etc
			time.Sleep(time.Until(nextPollTime))
		case !sm.IsActive():
			// here we don't really need it as there is no use of SlotMachine.Stop()
			return
		case sm.OccupiedSlotCount() < 2:
			// one player can't play, so we do a hard stop
			return
		default:
			// there are only SMs with WaitAny()
			// we have to wait for something external to happen
			wakeupSignal.Wait()
		}
	}
}
