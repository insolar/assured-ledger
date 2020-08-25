// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func simpleCtx() executionContext {
	slot := Slot{
		idAndStep: numberOfReservedSteps*stepIncrement | 1,
		slotData: slotData{
			asyncCallCount: 0,
		},
	}

	ctx := executionContext{
		slotContext:     slotContext{s: &slot},
		countAsyncCalls: 0,
		flags:           0,
	}
	ctx.slotContext.mode = updCtxExec

	return ctx
}

func TestExecutionContext_WaitActivation(t *testing.T) {
	testCases := []struct {
		name     string
		link     SlotLink
		decision Decision
	}{
		{name: "empty object", link: SlotLink{}},
		{name: "empty slot pointer", link: SlotLink{id: 1}, decision: NotPassed},
		{name: "available slot link", link: SlotLink{id: 1, s: &Slot{idAndStep: 1}}, decision: Passed},
		{name: "unavailable slot link", link: SlotLink{id: 1, s: &Slot{idAndStep: slotFlagBusy | 1}}, decision: NotPassed},
		{name: "wrong slot link id", link: SlotLink{id: 2, s: &Slot{idAndStep: 1}}, decision: Impossible},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(string); ok {
						assert.Equal(t, "illegal value", err)
					} else {
						t.Fatalf("unexpected error")
					}
				}

			}()
			ctx := simpleCtx()
			res := ctx.WaitActivation(testCase.link)
			assert.Equal(t, testCase.decision, res.GetDecision())
		})
	}
}

func TestExecutionContext_WaitShared(t *testing.T) {
	data := uniqueSharedKey{valueType: reflect.TypeOf("test")}
	testCases := []struct {
		name     string
		link     SharedDataLink
		decision Decision
	}{
		{name: "empty object", link: SharedDataLink{link: SlotLink{}}},
		{name: "empty data", link: SharedDataLink{link: SlotLink{id: 1}}, decision: NotPassed},
		{name: "empty slot pointer", link: SharedDataLink{link: SlotLink{id: 1}, data: data}, decision: NotPassed},
		{name: "available slot link", link: SharedDataLink{link: SlotLink{id: 1, s: &Slot{idAndStep: 1}}, data: data}, decision: Passed},
		{name: "unavailable slot link", link: SharedDataLink{link: SlotLink{id: 1, s: &Slot{idAndStep: slotFlagBusy | 1}}, data: data}, decision: NotPassed},
		{name: "wrong slot link id", link: SharedDataLink{link: SlotLink{id: 2, s: &Slot{idAndStep: 1}}, data: data}, decision: Impossible},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(string); ok {
						assert.Equal(t, "illegal value", err)
					} else {
						t.Fatalf("unexpected error")
					}
				}

			}()
			ctx := simpleCtx()
			res := ctx.WaitShared(testCase.link)
			assert.Equal(t, testCase.decision, res.GetDecision())
		})
	}
}
