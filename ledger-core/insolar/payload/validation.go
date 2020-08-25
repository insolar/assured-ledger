// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Validatable interface {
	Validate(currPulse PulseNumber) error
}

type PulseValidator interface {
	ValidateTime(pn pulse.Number, before pulse.Number) bool
	Relation() string
}

type ExtendedPulseValidator interface {
	PulseValidator
	GetSpecialValidator() PulseValidator
	ValidateSpecialOrTime(pn pulse.Number, before pulse.Number) bool
}

func isSpecialOrTimePulseBefore(pn pulse.Number, before pulse.Number) bool {
	return pn.IsSpecial() || (pn.IsTimePulse() && pn.IsBefore(before))
}

func isTimePulseBefore(pn pulse.Number, before pulse.Number) bool {
	return pn.IsTimePulse() && pn.IsBefore(before)
}

type timePulseBefore struct{}

func (timePulseBefore) GetSpecialValidator() PulseValidator {
	return specialOrTmePulseBefore{}
}

func (timePulseBefore) ValidateSpecialOrTime(pn pulse.Number, before pulse.Number) bool {
	return isSpecialOrTimePulseBefore(pn, before)
}

func (timePulseBefore) ValidateTime(pn pulse.Number, before pulse.Number) bool {
	return isTimePulseBefore(pn, before)
}
func (timePulseBefore) Relation() string { return "Before" }

type specialOrTmePulseBefore struct{}

func (specialOrTmePulseBefore) ValidateTime(pn pulse.Number, before pulse.Number) bool {
	return isSpecialOrTimePulseBeforeOrEq(pn, before)
}

func (specialOrTmePulseBefore) Relation() string { return "Before" }

func isSpecialOrTimePulseBeforeOrEq(pn pulse.Number, before pulse.Number) bool {
	return pn.IsSpecial() || (pn.IsTimePulse() && pn.IsBeforeOrEq(before))
}

func isTimePulseBeforeOrEq(pn pulse.Number, before pulse.Number) bool {
	return pn.IsTimePulse() && pn.IsBeforeOrEq(before)
}

type timePulseBeforeOrEq struct{}

func (timePulseBeforeOrEq) GetSpecialValidator() PulseValidator {
	return specialOrTmePulseBeforeOrEq{}
}

func (timePulseBeforeOrEq) ValidateSpecialOrTime(pn pulse.Number, before pulse.Number) bool {
	return isSpecialOrTimePulseBeforeOrEq(pn, before)
}

func (timePulseBeforeOrEq) ValidateTime(pn pulse.Number, before pulse.Number) bool {
	return isTimePulseBeforeOrEq(pn, before)
}

func (timePulseBeforeOrEq) Relation() string { return "BeforeOrEq" }

type specialOrTmePulseBeforeOrEq struct{}

func (specialOrTmePulseBeforeOrEq) ValidateTime(pn pulse.Number, before pulse.Number) bool {
	return isSpecialOrTimePulseBeforeOrEq(pn, before)
}

func (specialOrTmePulseBeforeOrEq) Relation() string { return "BeforeOrEq" }

func globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(global reference.Global) bool {
	var (
		basePulse  = global.GetBase().GetPulseNumber()
		localPulse = global.GetLocal().GetPulseNumber()
	)
	return basePulse.IsSpecial() || basePulse.IsBeforeOrEq(localPulse)
}

type fieldNameDescription struct {
	FieldName string
	Before    pulse.Number `opt:""`
	Relation  string       `opt:""`
	Base      pulse.Number `opt:""`
	Local     pulse.Number `opt:""`
}

func validSelfScopedGlobalWithPulseCheck(global reference.Global, before pulse.Number, fieldName string, checker PulseValidator) (pulse.Number, error) {
	if !global.IsSelfScope() {
		return pulse.Unknown, throw.New("Reference should be self scoped", fieldNameDescription{
			FieldName: fieldName,
		})
	}

	var (
		pn = global.GetLocal().Pulse()
	)

	if !checker.ValidateTime(pn, before) {
		return pulse.Unknown, throw.New("Reference pulse should be valid time pulse with relation to given pulse", fieldNameDescription{
			FieldName: fieldName,
			Local:     pn,
			Before:    before,
			Relation:  checker.Relation(),
		})
	}

	return pn, nil
}

func validSelfScopedGlobalWithPulseBefore(global reference.Global, before pulse.Number, fieldName string) (pulse.Number, error) {
	return validSelfScopedGlobalWithPulseCheck(global, before, fieldName, timePulseBefore{})
}

func validSelfScopedGlobalWithPulseBeforeOrEq(global reference.Global, before pulse.Number, fieldName string) (pulse.Number, error) {
	return validSelfScopedGlobalWithPulseCheck(global, before, fieldName, timePulseBeforeOrEq{})
}

func validSelfScopedGlobalWithPulseSpecialOrBefore(global reference.Global, before pulse.Number, fieldName string) (pulse.Number, error) {
	return validSelfScopedGlobalWithPulseCheck(global, before, fieldName, specialOrTmePulseBefore{})
}

func validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(global reference.Global, before pulse.Number, fieldName string) (pulse.Number, error) {
	return validSelfScopedGlobalWithPulseCheck(global, before, fieldName, specialOrTmePulseBeforeOrEq{})
}

func validRequestGlobalWithPulseCheck(outgoing reference.Global, before pulse.Number, fieldName string, checker ExtendedPulseValidator) (pulse.Number, error) {
	if outgoing.IsEmpty() {
		return pulse.Unknown, throw.New("Reference should be non-empty", fieldNameDescription{
			FieldName: fieldName,
		})
	}

	var (
		outgoingLocalPulse = outgoing.GetLocal().GetPulseNumber()
		outgoingBasePulse  = outgoing.GetBase().GetPulseNumber()
	)
	switch {
	case !checker.ValidateTime(outgoingLocalPulse, before):
		return pulse.Unknown, throw.New("Reference local pulse should be valid time pulse with relation to given pulse", fieldNameDescription{
			FieldName: fieldName,
			Local:     outgoingLocalPulse,
			Before:    before,
			Relation:  checker.Relation(),
		})
	case !checker.ValidateSpecialOrTime(outgoingBasePulse, before):
		// probably call outgoing base part can be special (API Call)
		return pulse.Unknown, throw.New("Base part should have valid pulse with relation to given pulse", fieldNameDescription{
			FieldName: fieldName,
			Base:      outgoingBasePulse,
			Before:    before,
			Relation:  checker.Relation(),
		})
	case !globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(outgoing):
		return pulse.Unknown, throw.New("Base pulse should be less or equal than local pulse", fieldNameDescription{
			FieldName: fieldName,
			Local:     outgoingLocalPulse,
			Base:      outgoingBasePulse,
			Relation:  checker.Relation(),
		})
	}

	return outgoingLocalPulse, nil
}

func validRequestGlobalWithPulseBefore(outgoing reference.Global, before pulse.Number, fieldName string) (pulse.Number, error) {
	return validRequestGlobalWithPulseCheck(outgoing, before, fieldName, timePulseBefore{})
}

func validRequestGlobalWithPulseBeforeOrEq(outgoing reference.Global, before pulse.Number, fieldName string) (pulse.Number, error) {
	return validRequestGlobalWithPulseCheck(outgoing, before, fieldName, timePulseBeforeOrEq{})
}

func validCallType(ct CallType) error {
	switch ct {
	case CallTypeMethod, CallTypeConstructor:
	case CallTypeInboundAPI, CallTypeOutboundAPI, CallTypeNotify, CallTypeSAGA, CallTypeParallel, CallTypeSchedule:
		return throw.New("CallType is not implemented")
	case CallTypeInvalid:
		fallthrough
	default:
		return throw.New("CallType must be valid")
	}

	return nil
}
