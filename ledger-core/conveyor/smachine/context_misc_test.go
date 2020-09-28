// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine_test

import (
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/stretchr/testify/assert"
)

type executionFuncType func (smachine.ExecutionContext) smachine.StateUpdate

type testSMFinalize struct {
	smachine.StateMachineDeclTemplate

	nFinalizeCalls    int
	executionFunc executionFuncType
}

func (s *testSMFinalize) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *testSMFinalize) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *testSMFinalize) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetFinalizer(s.finalize)
	return ctx.Jump(s.stepExecution2)
}

func (s *testSMFinalize) stepExecution1(ctx smachine.ExecutionContext) smachine.StateUpdate {
	panic(throw.IllegalState())
	return ctx.Stop()
}

func (s *testSMFinalize) stepExecution2(ctx smachine.ExecutionContext) smachine.StateUpdate {

	return ctx.Error(throw.New("Test error"))
}

func (s *testSMFinalize) finalize(ctx smachine.FinalizationContext) {
	s.nFinalizeCalls ++
	return
}

func TestSlotMachine_Finalize(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	scanCountLimit := 1000

	signal := synckit.NewVersionedSignal()
	m := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:    1000,
		PollingPeriod:   10 * time.Millisecond,
		PollingTruncate: 1 * time.Microsecond,
		ScanCountLimit:  scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	s := testSMFinalize{}
	s.executionFunc = s.stepExecution2
	m.AddNew(ctx, &s, smachine.CreateDefaultValues{})
	if !m.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}

	// make 1 iteration
	for {
		var (
			repeatNow bool
		)

		workerFactory.AttachTo(m, neverSignal, uint32(scanCountLimit), func(worker smachine.AttachedSlotWorker) {
			repeatNow, _ = m.ScanOnce(0, worker)
		})

		if repeatNow {
			continue
		}

		break
	}

	assert.Equal(t, 1, s.nFinalizeCalls)
	//assert.False(t, s.notOk)
}

/*
Kirill Ivkushkin:palm_tree: Sep 7th at 9:21 PM
к вопросу о defer'ах для SM - поперебирал варианты, и могу предложить один вариант, который вписывается в текущий функционал в пределах 100 строк кода, но имеет ряд функциональных ограничений. Подробнее см внутрь
24 replies

Kirill Ivkushkin:palm_tree:  16 days ago
возможности следующие:
на адаптерах можно сделать только SendNotify. Причина - даже если сделать Async, то некуда будет возвращать ответ, SM помрёт
можно работать с Publish / Unpublish / Share / Unshare, но нет доступа к TryUse
можно сделать NewChild / InitChild
можно сделать ReleaseAll и ApplyAdjustment

Kirill Ivkushkin:palm_tree:  16 days ago
остальной функционал будет недоступен

Kirill Ivkushkin:palm_tree:  16 days ago
да, и нельзя вернуть StateUpdate - т.е. "обратного пути" нет

Kirill Ivkushkin:palm_tree:  16 days ago
соотв выставлять такой хендлер надо будет как обычно, SetFinalizer
перекрыть его на уровне шага нельзя (через JumpExt)
вызываться он будет и после Stop и после ErrorHandler (если там явно не выставлен соотв флаг) (edited)

Kirill Ivkushkin:palm_tree:  16 days ago
вызываться он не будет - при останове SlotMachine и при вызове доп метода Halt
на работу Finalizer не распространяется действие ErrorHandler'а

Kirill Ivkushkin:palm_tree:  16 days ago
-----------

Kirill Ivkushkin:palm_tree:  16 days ago
вот как то так

Kirill Ivkushkin:palm_tree:  16 days ago
@eugene.blikh @ruslan.zakirov ^^
такой вариант надо?

Ruslan Zakirov  16 days ago
чет как-то дофига ограничений :slightly_smiling_face:

Kirill Ivkushkin:palm_tree:  16 days ago
ну дык это же останов SM
иначе задача решается локальной переменной в SM, которая передаётся в Jump, зачем это пихать в контекст то? :slightly_smiling_face: (edited)

Kirill Ivkushkin:palm_tree:  16 days ago
а так, для "финализации" функционал вполне достаточный

Eugene Blikh:press-f-2:  15 days ago
да, мне кажется это вполне подойдет для финализации по запросам, других кейсов я пока не вижу

Kirill Ivkushkin:palm_tree:  15 days ago
ок, значит сделаю

Kirill Ivkushkin:palm_tree:  15 days ago
для простой SMки уложился в 100-200 строк, но вот обработка ошибок для субрутин оказалась плохо совместимой, так что без моков строк на 500 вышло
https://github.com/insolar/assured-ledger/pull/752

cyraxredcyraxred
#752 NOISSUE: Finalizer for SM
• Finalizer handler for SM
• Refactoring
• Changed subroutine and error handling to support finalizer
Comments
1
<https://github.com/insolar/assured-ledger|insolar/assured-ledger>insolar/assured-ledger | Sep 8th | Added by GitHub

Kirill Ivkushkin:palm_tree:  15 days ago
@eugene.blikh @ruslan.zakirov ^^

Kirill Ivkushkin:palm_tree:  15 days ago
@ruslan.zakirov а есть возможность плз кого-то из SDETов попросить написать тесты на finalizer? что он срабатывает при stop, error, panic как из основной SM, так и из субрутин

Ruslan Zakirov  14 days ago
ок

Ruslan Zakirov  13 days ago
https://insolar.atlassian.net/browse/PLAT-820

PLAT-820 Тесты финализатора SMок
Status: To Do
Type: Task
Assignee: Maria Vasilenko
Priority: Medium
Added by Jira Cloud

Kirill Ivkushkin:palm_tree:  13 days ago
спасибо!

Ruslan Zakirov  13 days ago
у нас есть тест, который можно c&p как основу? (edited)

Kirill Ivkushkin:palm_tree:  13 days ago
TestSlotMachine_AddSMAndMigrate
например
тока причесать надо, чтобы создание SlotMachine было через метод

Kirill Ivkushkin:palm_tree:  13 days ago
т.е. тест объявляет свои SM, запускает их исполнение, и смотрит, что вышло, через флаги
(если цикл исполнения внутри метода теста (как в примере выше), или через атомик-счётчики,
если исполнение/worker внешний (edited)

Ruslan Zakirov  13 days ago
спасибо

Kirill Ivkushkin:palm_tree:  13 days ago
welcome
 */
