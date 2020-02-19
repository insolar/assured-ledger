// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"fmt"
	"runtime/debug"
)

var _ error = SlotPanicError{}

type SlotPanicError struct {
	Msg       string
	Recovered interface{}
	Prev      error
	Stack     []byte
	IsAsync   bool
}

func (e SlotPanicError) Error() string {
	sep := ""
	if len(e.Stack) > 0 {
		sep = "\n"
	}
	if e.Prev != nil {
		return fmt.Sprintf("%s: %v%s%s\nCaused by:\n%s", e.Msg, e.Recovered, sep, string(e.Stack), e.Prev.Error())
	}
	return fmt.Sprintf("%s: %v%s%s", e.Msg, e.Recovered, sep, string(e.Stack))
}

func (e SlotPanicError) String() string {
	return fmt.Sprintf("%s: %v", e.Msg, e.Recovered)
}

func RecoverSlotPanic(msg string, recovered interface{}, prev error) error {
	if recovered == nil {
		return prev
	}
	return SlotPanicError{Msg: msg, Recovered: recovered, Prev: prev}
}

func RecoverSlotPanicWithStack(msg string, recovered interface{}, prev error) error {
	if recovered == nil {
		return prev
	}
	return SlotPanicError{Msg: msg, Recovered: recovered, Prev: prev, Stack: debug.Stack()}
}

func RecoverAsyncSlotPanicWithStack(msg string, recovered interface{}, prev error) error {
	if recovered == nil {
		return prev
	}
	return SlotPanicError{Msg: msg, Recovered: recovered, Prev: prev, Stack: debug.Stack(), IsAsync: true}
}
