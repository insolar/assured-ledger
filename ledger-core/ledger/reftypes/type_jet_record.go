package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func JetRecordRef(id jet.ID, recordRef reference.LocalHolder) reference.Global {
	if id == 0 {
		panic(throw.IllegalValue())
	}
	local := recordRef.GetLocal()
	if !pulseZeroScope(local.GetHeader()).IsTimePulse() {
		panic(throw.IllegalValue())
	}
	return reference.New(JetLocalRef(id), local)
}

func JetRecordRefByDrop(id jet.DropID, recordRef reference.LocalHolder) reference.Global {
	local := recordRef.GetLocal()
	if id.CreatedAt() != local.Pulse() {
		panic(throw.IllegalValue())
	}
	return JetRecordRef(id.ID(), local)
}

func UnpackJetRecordRef(ref reference.Holder) (jet.DropID, reference.Local, error) {
	if ref == nil {
		return 0, reference.Local{}, newRefTypeErr(ErrIllegalRefValue, JetRecord, reference.Local{}, reference.Local{})
	}

	base, local := ref.GetBase(), ref.GetLocal()
	lpn := local.Pulse()
	if pulseZeroScope(base.GetHeader()) == pulse.Jet && lpn.IsTimePulse() {
		pn, id, err := DecodeJetData(base.IdentityHash(), true)

		switch {
		case err != nil:
		case pn != pulse.Unknown || id.HasLength():
			// wrong subtype
			err = ErrInvalidJetRecordRef
		default:
			return id.AsDrop(lpn), local, nil
		}
		return 0, reference.Local{}, newRefTypeErr(err, JetRecord, base, local)
	}
	return 0, reference.Local{}, newRefTypeErr(ErrIllegalRefValue, JetRecord, base, local)
}

func JetRecordOf(ref reference.Holder) reference.Global {
	id, local, err := UnpackAsJetRecordOf(ref)
	if err != nil {
		return reference.Global{}
	}
	return JetRecordRef(id.ID(), local)
}

func UnpackAsJetRecordOf(ref reference.Holder) (jet.DropID, reference.Local, error) {
	if ref == nil {
		return 0, reference.Local{}, ErrIllegalRefValue
	}

	base, local := ref.GetBase(), ref.GetLocal()

	lpn := local.Pulse()
	if !lpn.IsTimePulse() {
		return 0, reference.Local{}, ErrIllegalRefValue
	}

	pn := pulse.Unknown
	id := jet.UnknownExactID
	var err error

	switch pulseZeroScope(base.GetHeader()) {
	case pulse.Jet:
		pn, id, err = DecodeJetData(base.IdentityHash(), true)
	case pulse.RecordPayload:
		data := base.IdentityHash()
		pn, id, err = DecodeJetData(data, false)
		if err == nil {
			_, _, _, err = DecodeJetPayloadPosition(data, true)
		}
	default:
		err = ErrIllegalRefValue
	}

	switch {
	case err != nil:
		return 0, reference.Local{}, err
	case pn == pulse.Unknown:
		// jet ref
	case id.HasLength():
		// jet leg
		if pn > lpn {
			return 0, reference.Local{}, ErrInvalidJetRecordRef
		}
	case pn == lpn:
	default:
		return 0, reference.Local{}, ErrInvalidJetRecordRef
	}

	return id.AsDrop(lpn), local, nil
}

/**********************************************/

var _ RefTypeDef = typeDefJetRecord{}

type typeDefJetRecord struct{}

func (typeDefJetRecord) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefJetRecord) Usage() Usage {
	return UseAsBase
}

func (v typeDefJetRecord) RefFrom(base, local reference.Local) (reference.Global, error) {
	if err := v.VerifyGlobalRef(base, local); err != nil {
		return reference.Global{}, err
	}
	return reference.New(base, local), nil
}

func (typeDefJetRecord) VerifyGlobalRef(base, local reference.Local) error {
	_, _, err := UnpackJetRecordRef(reference.New(base, local))
	return err
}

func (typeDefJetRecord) VerifyLocalRef(ref reference.Local) error {
	panic(throw.Unsupported())
}

func (typeDefJetRecord) DetectSubType(_, _ reference.Local) RefType {
	panic(throw.Unsupported()) // it is a sub type
}
