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

type SlotPanicArea uint8

const (
	_ SlotPanicArea = iota
	InternalArea
	ErrorHandlerArea
	StateArea
	SubroutineArea
	BargeInArea
	AsyncCallArea
)

func (v SlotPanicArea) IsDetached() bool {
	return v >= BargeInArea
}

func (v SlotPanicArea) CanRecoverByHandler() bool {
	return v.IsDetached()
}

func (v SlotPanicArea) CanRecoverBySubroutine() bool {
	return v >= ErrorHandlerArea
}

type SlotPanicError struct {
	Msg       string
	Recovered interface{}
	Prev      error
	Stack     []byte
	Area      SlotPanicArea
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

func RecoverSlotPanic(msg string, recovered interface{}, prev error, area SlotPanicArea) error {
	if recovered == nil {
		return prev
	}
	return SlotPanicError{Msg: msg, Recovered: recovered, Prev: prev, Area: area}
}

func RecoverSlotPanicWithStack(msg string, recovered interface{}, prev error, area SlotPanicArea) error {
	if recovered == nil {
		return prev
	}
	return SlotPanicError{Msg: msg, Recovered: recovered, Prev: prev, Area: area, Stack: debug.Stack()}
}
