// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	ErrTimePulseRequired        = throw.W(ErrIllegalRefValue, "time-pulse required")
	ErrScopeNotImplemented      = throw.W(ErrIllegalRefValue, "scopes are not implemented")
	ErrTimePulseNoScopeRequired = throw.W(ErrIllegalRefValue, "time-pulse with lifeline scope is required")
	ErrLocalBeforeBase          = throw.W(ErrIllegalRefValue, "local pulse is before base pulse")
)

func LifelineRef(base reference.Local) reference.Global {
	switch {
	case !base.Pulse().IsTimePulse():
		panic(throw.IllegalValue())
	case base.SubScope() != 0:
		panic(throw.NotImplemented())
	}
	return reference.NewSelf(base)
}

func SidelineRef(lifeline reference.Holder, loc reference.Local) reference.Global {
	base, local := lifeline.GetBase(), loc.GetLocal()
	ensureObjRef(base, local)

	if base != lifeline.GetLocal() {
		panic(throw.IllegalValue())
	}
	return reference.New(base, local)
}

func RecordRef(line reference.Holder, loc reference.Local) reference.Global {
	base, local := line.GetBase(), loc.GetLocal()
	ensureObjRef(base, local)
	ensureObjRef(base, line.GetLocal())
	return reference.New(base, local)
}

func ensureObjRef(base, local reference.Local) {
	switch {
	case !base.Pulse().IsTimePulse():
		panic(throw.IllegalValue())
	case base.SubScope() != 0:
		panic(throw.NotImplemented())
	case !pulseZeroScope(local.GetHeader()).IsTimePulse():
		panic(throw.IllegalValue())
	case base.Pulse() < local.Pulse():
		panic(throw.IllegalValue())
	}
}

func inspectObjRef(base, local reference.Local) error {
	switch {
	case !base.Pulse().IsTimePulse():
		return ErrTimePulseRequired
	case base.SubScope() != 0:
		return ErrScopeNotImplemented
	case !pulseZeroScope(local.GetHeader()).IsTimePulse():
		return ErrTimePulseNoScopeRequired
	case base.Pulse() < local.Pulse():
		return ErrLocalBeforeBase
	}
	return nil
}

func LifelineRefOf(ref reference.Holder) reference.Global {
	return reference.NewSelf(LifelineLocalRefOf(ref))
}

func LifelineLocalRefOf(ref reference.Holder) reference.Local {
	base, local := ref.GetBase(), ref.GetLocal()
	if inspectObjRef(base, local) == nil {
		return base
	}
	return reference.Local{}
}

func UnpackObjectRef(ref reference.Holder) (base, local reference.Local, err error) {
	base, local = ref.GetBase(), ref.GetLocal()
	if err := inspectObjRef(base, local); err != nil {
		return reference.Local{}, reference.Local{}, newRefTypeErr(err, Object, base, local)
	}
	return base, local, nil
}

func UnpackObjectLocalRef(ref reference.Local) (reference.Local, error) {
	local := ref.GetLocal()
	if err := inspectObjRef(local, local); err != nil {
		return reference.Local{}, newRefTypeErr(err, Object, reference.Local{}, local)
	}
	return local, nil
}

/*****************************************************/

var _ RefTypeDef = typeDefObject{}

type typeDefObject struct{}

func (typeDefObject) CanBeDerivedWith(pn pulse.Number, local reference.Local) bool {
	lpn := pulseZeroScope(local.GetHeader())
	return lpn >= pn && lpn.IsTimePulse()
}

func (v typeDefObject) RefFrom(base, local reference.Local) (reference.Global, error) {
	if err := v.VerifyGlobalRef(base, local); err != nil {
		return reference.Global{}, err
	}
	return reference.New(base, local), nil
}

func (typeDefObject) Usage() Usage {
	return UseAsBase | UseAsSelf | UseAsLocalValue
}

func (typeDefObject) VerifyGlobalRef(base, local reference.Local) error {
	_, _, err := UnpackObjectRef(reference.New(base, local))
	return err
}

func (typeDefObject) VerifyLocalRef(ref reference.Local) error {
	_, err := UnpackObjectLocalRef(ref)
	return err
}

func (typeDefObject) DetectSubType(_, _ reference.Local) RefType {
	return 0
}
