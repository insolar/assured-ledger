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
		name string
		link SlotLink

		expectedDecision Decision
		expectedError    bool
	}{
		{name: "empty object", link: SlotLink{}, expectedError: true},
		{name: "empty slot pointer", link: SlotLink{id: 1}, expectedError: true},
		{name: "available slot link", link: SlotLink{id: 1, s: &Slot{idAndStep: 1}}, expectedDecision: Passed},
		{name: "unavailable slot link", link: SlotLink{id: 1, s: &Slot{idAndStep: slotFlagBusy | 1}}, expectedDecision: NotPassed},
		{name: "wrong slot link id", link: SlotLink{id: 2, s: &Slot{idAndStep: 1}}, expectedDecision: Impossible},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.expectedError {
				defer func() {
					assert.EqualError(t, recover().(error), "illegal value")
				}()
			}

			ctx := simpleCtx()
			res := ctx.WaitActivation(testCase.link)
			assert.Equal(t, testCase.expectedDecision, res.GetDecision())
		})
	}
}

func TestExecutionContext_WaitShared(t *testing.T) {
	data := uniqueSharedKey{valueType: reflect.TypeOf("test")}
	testCases := []struct {
		name string
		link SharedDataLink

		expectedDecision Decision
		expectedError    bool
	}{
		{name: "empty object", link: SharedDataLink{link: SlotLink{}}, expectedError: true},
		{name: "empty data", link: SharedDataLink{link: SlotLink{id: 1}}, expectedError: true},
		{name: "empty slot pointer", link: SharedDataLink{link: SlotLink{id: 1}, data: data}, expectedError: true},
		{name: "available slot link", link: SharedDataLink{link: SlotLink{id: 1, s: &Slot{idAndStep: 1}}, data: data}, expectedDecision: Passed},
		{name: "unavailable slot link", link: SharedDataLink{link: SlotLink{id: 1, s: &Slot{idAndStep: slotFlagBusy | 1}}, data: data}, expectedDecision: NotPassed},
		{name: "wrong slot link id", link: SharedDataLink{link: SlotLink{id: 2, s: &Slot{idAndStep: 1}}, data: data}, expectedDecision: Impossible},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.expectedError {
				defer func() {
					assert.EqualError(t, recover().(error), "illegal value")
				}()
			}

			ctx := simpleCtx()
			res := ctx.WaitShared(testCase.link)
			assert.Equal(t, testCase.expectedDecision, res.GetDecision())
		})
	}
}
