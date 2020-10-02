// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ error = SlotPanicError{}

type SlotPanicArea uint8

const (
	_ SlotPanicArea = iota
	InternalArea
	ErrorHandlerArea
	FinalizerArea
	StepArea
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

func (v SlotPanicArea) CanRecoverBySubroutineExit() bool {
	return v >= ErrorHandlerArea
}

type SlotPanicError struct {
	Msg       string
	Recovered interface{}
	Prev      error
	Area      SlotPanicArea
}

func (e SlotPanicError) Error() string {
	if e.Prev != nil {
		return fmt.Sprintf("%s: %v\nCaused by:\n%s", e.Msg, e.Recovered, e.Prev.Error())
	}
	return fmt.Sprintf("%s: %v", e.Msg, e.Recovered)
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
	return throw.WithStackExt(SlotPanicError{Msg: msg, Recovered: recovered, Prev: prev, Area: area}, 1)
}
