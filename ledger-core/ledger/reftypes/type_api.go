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

func APICallRef(nodeRef reference.Holder, sidPN pulse.Number, reqHash reference.LocalHash) reference.Global {
	if !sidPN.IsTimePulse() {
		panic(throw.IllegalValue())
	}

	base, err := UnpackNodeRefAsLocal(nodeRef)
	if err != nil {
		panic(err)
	}

	apiBase := reference.NewLocal(pulse.ExternalCall, 0, base.IdentityHash())
	return reference.New(apiBase, reference.NewLocal(sidPN, 0, reqHash))
}

func UnpackAPICallRef(ref reference.Holder) (nodeRef reference.Global, sidPN pulse.Number, reqHash reference.LocalHash, err error) {
	base, local := ref.GetBase(), ref.GetLocal()
	lpn := local.Pulse()
	switch {
	case !lpn.IsTimePulse():
		err = ErrIllegalRefValue
	case pulse.ExternalCall != pulseZeroScope(base.GetHeader()):
		err = ErrIllegalRefValue
	default:
		return NodeRef(base.IdentityHash()), lpn, local.IdentityHash(), nil
	}

	err = newRefTypeErr(err, APICall, base, local)
	return reference.Global{}, 0, reference.LocalHash{}, err
}

/*****************************************************/

var _ RefTypeDef = typeDefAPICall{}
type typeDefAPICall struct {}

func (typeDefAPICall) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefAPICall) Usage() Usage {
	return UseAsBase
}

func (v typeDefAPICall) RefFrom(base, local reference.Local) (reference.Global, error) {
	if err := v.VerifyGlobalRef(base, local); err != nil {
		return reference.Global{}, err
	}
	return reference.New(base, local), nil
}

func (typeDefAPICall) VerifyGlobalRef(base, local reference.Local) error {
	_, _, _, err := UnpackAPICallRef(reference.New(base, local))
	return err
}

func (typeDefAPICall) VerifyLocalRef(reference.Local) error {
	panic(throw.Unsupported())
}

func (typeDefAPICall) DetectSubType(_, _ reference.Local) RefType {
	return 0 // no subtypes
}
