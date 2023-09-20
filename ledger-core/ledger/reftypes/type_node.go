package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NodeLocalRef(nodeHash reference.LocalHash) reference.Local {
	return reference.NewLocal(pulse.Node, 0, nodeHash)
}

func NodeRef(nodeHash reference.LocalHash) reference.Global {
	return reference.NewSelf(NodeLocalRef(nodeHash))
}

func NodeContractRef(nodeHash reference.LocalHash, locContract reference.LocalHolder) reference.Global {
	local := locContract.GetLocal()
	if !local.Pulse().IsTimePulse() {
		panic(throw.IllegalValue())
	}
	return reference.New(NodeLocalRef(nodeHash), local)
}

func nodeLocalRefOf(loc reference.LocalHolder) (pulse.Number, reference.Local) {
	local := loc.GetLocal()
	switch pn := pulseZeroScope(local.GetHeader()); pn {
	case pulse.Node:
		return pn, local
	case pulse.ExternalCall:
		return pn, NodeLocalRef(local.IdentityHash())
	default:
		return pn, reference.Local{}
	}
}

func NodeLocalRefOf(ref reference.Holder) reference.Local {
	pn, base := nodeLocalRefOf(ref.GetBase())
	local := ref.GetLocal()
	switch {
	case pn != pulse.Node:
		// api call or empty
	case local.Pulse().IsTimePulse():
		// node contract
	case base != local:
		// otherwise it must be self ref
		return reference.Local{}
	}
	return base
}

func UnpackNodeRefAsLocal(ref reference.Holder) (nodeRef reference.Local, err error) {
	pn, base := nodeLocalRefOf(ref.GetBase())
	local := ref.GetLocal()
	switch {
	case pn != pulse.Node:
		err = ErrIllegalRefValue
	case base != ref.GetLocal():
		// not a node ref, may be a contract ref
		err = ErrIllegalSelfRefValue
	default:
		return base, nil
	}
	err = newRefTypeErr(err, Node, base, local)
	return reference.Local{}, err
}

func UnpackNodeRef(ref reference.Holder) (reference.Global, error) {
	base, err := UnpackNodeRefAsLocal(ref)
	if err != nil {
		return reference.Global{}, err
	}
	return reference.NewSelf(base), nil
}

func UnpackNodeLocalRef(ref reference.LocalHolder) (reference.Local, error) {
	pn, local := nodeLocalRefOf(ref.GetLocal())
	switch {
	case pn != pulse.Node:
		return reference.Local{}, newRefTypeErr(ErrIllegalRefValue, Node, reference.Local{}, local)
	default:
		return local, nil
	}
}

func UnpackNodeContractRef(ref reference.Holder) (nodeRef reference.Global, contractRef reference.Local, err error) {
	pn, base := nodeLocalRefOf(ref.GetBase())
	local := ref.GetLocal()
	switch {
	case pn != pulse.Node:
		err = ErrIllegalRefValue
	case local.Pulse().IsTimePulse():
		return reference.NewSelf(base), local, nil
	default:
		err = ErrIllegalRefValue
	}

	err = newRefTypeErr(err, NodeContract, base, local)
	return reference.Global{}, reference.Local{}, err
}

/*****************************************************/

var _ RefTypeDef = typeDefNode{}
type typeDefNode struct {}

func (typeDefNode) CanBeDerivedWith(_ pulse.Number, local reference.Local) bool {
	return local.Pulse().IsTimePulse()
}

func (typeDefNode) RefFrom(base, local reference.Local) (reference.Global, error) {
	if base := NodeLocalRefOf(reference.New(base, local)); !base.IsEmpty() {
		return reference.NewSelf(base), nil
	}
	return reference.Global{}, newRefTypeErr(ErrIllegalRefValue, Node, base, local)
}

func (typeDefNode) Usage() Usage {
	return UseAsBase|UseAsSelf|UseAsLocalValue
}

func (typeDefNode) VerifyGlobalRef(base, local reference.Local) error {
	_, err := UnpackNodeRef(reference.New(base, local))
	return err
}

func (typeDefNode) VerifyLocalRef(ref reference.Local) error {
	_, err := UnpackNodeLocalRef(ref)
	return err
}

func (typeDefNode) DetectSubType(base, local reference.Local) RefType {
	switch {
	case base == local:
		return Node
	case local.Pulse().IsTimePulse():
		return NodeContract
	}
	return Invalid
}

var _ RefTypeDef = typeDefNodeContract{}
type typeDefNodeContract struct {}

func (typeDefNodeContract) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefNodeContract) Usage() Usage {
	return UseAsBase
}

func (v typeDefNodeContract) RefFrom(base, local reference.Local) (reference.Global, error) {
	if err := v.VerifyGlobalRef(base, local); err != nil {
		return reference.Global{}, err
	}
	return reference.New(base, local), nil
}

func (typeDefNodeContract) VerifyGlobalRef(base, local reference.Local) error {
	_, _, err := UnpackNodeContractRef(reference.New(base, local))
	return err
}

func (typeDefNodeContract) VerifyLocalRef(ref reference.Local) error {
	panic(throw.Unsupported())
}

func (typeDefNodeContract) DetectSubType(_, _ reference.Local) RefType {
	panic(throw.Unsupported()) // it is a sub type
}
