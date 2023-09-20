package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type BuiltinContractType uint8

const (
	UnknownBuiltin BuiltinContractType = 0
	ExternalNodeAPI BuiltinContractType = 0xC2 // This constant allows textual refs to start with: 0AAABApi[A..P]
)

type BuiltinContractID [27]byte

func BuiltinContractLocalRef(t BuiltinContractType, primaryID BuiltinContractID) reference.Local {
	if t == UnknownBuiltin {
		panic(throw.IllegalValue())
	}

	var data reference.LocalHash
	data[0] = byte(t)
	if copy(data[1:], primaryID[:]) != len(primaryID) {
		panic(throw.Impossible())
	}

	return reference.NewLocal(pulse.BuiltinContract, 0, data)
}

func BuiltinContractRef(t BuiltinContractType, primaryID BuiltinContractID) reference.Global {
	return reference.NewSelf(BuiltinContractLocalRef(t, primaryID))
}

func UnpackBuiltinContractRef(ref reference.Holder) (t BuiltinContractType, primaryID BuiltinContractID, secondaryID reference.LocalHolder, err error) {
	base, local := ref.GetBase(), ref.GetLocal()
	switch {
	case pulseZeroScope(base.GetHeader()) != pulse.BuiltinContract:
	case local.IsEmpty():
	default:
		data := base.IdentityHash()
		t = BuiltinContractType(data[0])
		if t == UnknownBuiltin {
			break
		}
		if copy(primaryID[:], data[1:]) != len(primaryID) {
			panic(throw.Impossible())
		}

		if base != local {
			return t, primaryID, local, nil
		}
		return t, primaryID, nil, nil
	}
	err = newRefTypeErr(ErrIllegalRefValue, BuiltinContract, base, local)
	return 0, BuiltinContractID{}, nil, err
}

/*****************************************************/

var _ RefTypeDef = typeDefBuiltinContract{}
type typeDefBuiltinContract struct {}

func (typeDefBuiltinContract) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefBuiltinContract) Usage() Usage {
	return UseAsBase
}

func (v typeDefBuiltinContract) RefFrom(base, local reference.Local) (reference.Global, error) {
	if err := v.VerifyGlobalRef(base, local); err != nil {
		return reference.Global{}, err
	}
	return reference.New(base, local), nil
}

func (typeDefBuiltinContract) VerifyGlobalRef(base, local reference.Local) error {
	_, _, _, err := UnpackBuiltinContractRef(reference.New(base, local))
	return err
}

func (typeDefBuiltinContract) VerifyLocalRef(reference.Local) error {
	panic(throw.Unsupported())
}

func (typeDefBuiltinContract) DetectSubType(_, _ reference.Local) RefType {
	return 0 // no subtypes
}
