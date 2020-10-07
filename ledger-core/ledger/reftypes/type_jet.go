// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"encoding/binary"

	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	ErrInvalidJetDropRef   = throw.W(ErrIllegalRefValue, "jet drop/leg id is invalid")
	ErrUnexpectedJetRef    = throw.W(ErrIllegalRefValue, "unexpected jet drop/leg ref")
	ErrInvalidJetRecordRef = throw.W(ErrIllegalRefValue, "invalid jet record base")
)


func JetLocalRef(id jet.ID) reference.Local {
	var data reference.LocalHash
	EncodeJetData(&data, 0, jet.NoLengthExactID(id))
	return reference.NewLocal(pulse.Jet, 0, data)
}

func JetRef(id jet.ID) reference.Global {
	return reference.NewSelf(JetLocalRef(id))
}

func JetLocalRefOf(ref reference.Holder) reference.Local {
	switch pn, id := jetDropLocalRefOf(ref, false); {
	case pn == invalidJetDropRef:
		return reference.Local{}
	default:
		return JetLocalRef(id)
	}
}

func UnpackJetLocalRef(ref reference.LocalHolder) (jet.ID, error) {
	local := ref.GetLocal()
	switch id, err := unpackJetLocalRef(local, true); {
	case err != nil:
		return 0, newRefTypeErr(err, Jet, reference.Local{}, local)
	default:
		return id, nil
	}
}

func PartialUnpackJetLocalRef(ref reference.LocalHolder) (jet.ID, error) {
	local := ref.GetLocal()
	switch id, err := unpackJetLocalRef(local, false); {
	case err != nil:
		return 0, newRefTypeErr(err, Jet, reference.Local{}, local)
	default:
		return id, nil
	}
}

func UnpackJetRef(ref reference.Holder) (jet.ID, error) {
	base, local := ref.GetBase(), ref.GetLocal()
	switch id, err := unpackJetLocalRef(base, true); {
	case err != nil:
		return 0, newRefTypeErr(err, Jet, base, local)
	case base != local:
		return 0, newRefTypeErr(ErrIllegalSelfRefValue, Jet, base, local)
	default:
		return id, nil
	}
}

func unpackJetLocalRef(local reference.Local, checkTailingZeros bool) (jet.ID, error) {
	if pulseZeroScope(local.GetHeader()) != pulse.Jet {
		return 0, ErrIllegalRefValue
	}

	pn, id, err := DecodeJetData(local.IdentityHash(), checkTailingZeros)
	switch {
	case err != nil:
		return 0, err
	case pn	!= 0:
		return 0, ErrUnexpectedJetRef
	case id.HasLength():
		return 0, ErrIllegalRefValue
	default:
		return id.ID(), nil
	}
}

func JetLegLocalRef(id jet.LegID) reference.Local {
	switch {
	case !id.CreatedAt().IsTimePulse():
		panic(throw.IllegalState())
	case !id.ExactID().HasLength():
		panic(throw.IllegalState())
	}

	var data reference.LocalHash
	EncodeJetData(&data, id.CreatedAt(), id.ExactID())
	return reference.NewLocal(pulse.Jet, 0, data)
}

func JetLegRef(id jet.LegID) reference.Global {
	return reference.NewSelf(JetLegLocalRef(id))
}

func UnpackJetLegLocalRef(ref reference.LocalHolder) (jet.LegID, error) {
	local := ref.GetLocal()
	switch pn, id, err := unpackJetDropLocalRef(local, true); {
	case err != nil:
		return 0, newRefTypeErr(err, JetLeg, reference.Local{}, local)
	default:
		return id.AsLeg(pn), nil
	}
}

func UnpackJetLegRef(ref reference.Holder) (jet.LegID, error) {
	base, local := ref.GetBase(), ref.GetLocal()
	switch pn, id, err := unpackJetDropLocalRef(base, true); {
	case err != nil:
		return 0, newRefTypeErr(err, JetLeg, base, local)
	case base != local:
		return 0, newRefTypeErr(ErrIllegalSelfRefValue, JetLeg, base, local)
	default:
		return id.AsLeg(pn), nil
	}
}

func JetDropLocalRef(id jet.DropID) reference.Local {
	if !id.CreatedAt().IsTimePulse() {
		panic(throw.IllegalState())
	}

	var data reference.LocalHash
	EncodeJetData(&data, id.CreatedAt(), jet.NoLengthExactID(id.ID()))
	return reference.NewLocal(pulse.Jet, 0, data)
}

func JetDropRef(id jet.DropID) reference.Global {
	return reference.NewSelf(JetDropLocalRef(id))
}

func JetDropLocalRefOf(ref reference.Holder) reference.Local {
	switch pn, id := jetDropLocalRefOf(ref, true); {
	case pn <= invalidJetDropRef:
		return reference.Local{}
	default:
		return JetDropLocalRef(id.AsDrop(pn))
	}
}

const invalidJetDropRef = pulse.Number(1)

func jetDropLocalRefOf(ref reference.Holder, mustBeDrop bool) (pulse.Number, jet.ID) {
	if ref == nil {
		return invalidJetDropRef, 0
	}

	base, local := ref.GetBase(), ref.GetLocal()
	switch pulseZeroScope(base.GetHeader()) {
	case pulse.Jet:
		if base == local {
			// self ref
			break
		}
		if !local.Pulse().IsTimePulse() {
			return invalidJetDropRef, 0
		}
	case pulse.RecordPayload:
		if !local.Pulse().IsTimePulse() {
			return invalidJetDropRef, 0
		}
		mustBeDrop = true
	default:
		return invalidJetDropRef, 0
	}

	pn, id := jetDropOfLocalRef(base, mustBeDrop)
	return pn, id.ID()
}

func jetDropOfLocalRef(base reference.Local, mustBeDrop bool) (pulse.Number, jet.ExactID) {
	pn, id, err := DecodeJetData(base.IdentityHash(), false)
	switch {
	case err != nil:
		return invalidJetDropRef, 0
	case pn	== 0:
		// it is a jet ref
		if mustBeDrop {
			return invalidJetDropRef, 0
		}
	case mustBeDrop == id.HasLength():
		// it is a jet leg ref
		return invalidJetDropRef, 0
	}
	return pn, id
}

func UnpackJetDropLocalRef(ref reference.LocalHolder) (jet.DropID, error) {
	local := ref.GetLocal()
	switch pn, id, err := unpackJetDropLocalRef(local, false); {
	case err != nil:
		return 0, newRefTypeErr(err, JetDrop, reference.Local{}, local)
	default:
		return id.AsDrop(pn), nil
	}
}

func UnpackJetDropRef(ref reference.Holder) (jet.DropID, error) {
	base, local := ref.GetBase(), ref.GetLocal()
	switch pn, id, err := unpackJetDropLocalRef(base, false); {
	case err != nil:
		return 0, newRefTypeErr(err, JetDrop, base, local)
	case base != local:
		return 0, newRefTypeErr(ErrIllegalSelfRefValue, JetDrop, base, local)
	default:
		return id.AsDrop(pn), nil
	}
}

func unpackJetDropLocalRef(local reference.Local, checkLeg bool) (pulse.Number, jet.ExactID, error) {
	if pulseZeroScope(local.GetHeader()) != pulse.Jet {
		return 0, 0, ErrIllegalRefValue
	}

	pn, id, err := DecodeJetData(local.IdentityHash(), true)
	switch {
	case err != nil:
		return 0, 0, err
	case !pn.IsTimePulse():
		return 0, 0, ErrInvalidJetDropRef
	case checkLeg != id.HasLength():
		return 0, 0, ErrInvalidJetDropRef
	default:
		return pn, id, nil
	}
}

const jetDataLength = 8

func DecodeJetData(b reference.LocalHash, checkTailingZeros bool) (pulse.Number, jet.ExactID, error) {
	encoder := binary.LittleEndian
	pnn := encoder.Uint32(b[:4])
	jetID := encoder.Uint16(b[4:6])
	if b[6] != 0 {
		return 0, 0, throw.W(ErrIllegalRefValue,"invalid reserved[6] in jet ref")
	}
	pfxLen := b[:jetDataLength][7] // just a sanity check

	if checkTailingZeros {
		for _, bi := range b[jetDataLength:] {
			if bi != 0 {
				return 0, 0, throw.W(ErrIllegalRefValue,"unexpected data in jet ref", struct { PN uint32 }{ pnn })
			}
		}
	}

	if pnn != 0 && !pulse.IsValidAsPulseNumber(int(pnn)) {
		return 0, 0, throw.W(ErrIllegalRefValue,"invalid pulse number in jet ref", struct { PN uint32 }{ pnn })
	}
	pn := pulse.Number(pnn)

	switch {
	case pfxLen == 0:
		return pn, jet.NoLengthExactID(jet.ID(jetID)), nil
	case pfxLen > jet.IDBitLen:
		return 0, 0, throw.W(ErrIllegalRefValue,"invalid prefix length in jet ref")
	case pn == 0:
		return 0, 0, throw.W(ErrIllegalRefValue,"missing pulse in jet leg ref")
	default:
		return pn, jet.ID(jetID).AsExact(pfxLen), nil
	}
}

func EncodeJetData(b *reference.LocalHash, pn pulse.Number, id jet.ExactID) {
	if !pn.IsUnknownOrTimePulse() {
		panic(throw.IllegalValue())
	}

	encoder := binary.LittleEndian
	encoder.PutUint32(b[:4], uint32(pn))
	encoder.PutUint16(b[4:6], uint16(id.ID()))
	b[6] = 0
	if id.HasLength() {
		b[7] = id.BitLen()
	} else {
		b[7] = 0
	}
}

/*****************************************************/

var _ RefTypeDef = typeDefJet{}
type typeDefJet struct {}

func (typeDefJet) CanBeDerivedWith(_ pulse.Number, local reference.Local) bool {
	return pulseZeroScope(local.GetHeader()).IsTimePulse()
}

func (typeDefJet) Usage() Usage {
	return UseAsBase|UseAsSelf|UseAsLocalValue
}

func (typeDefJet) RefFrom(base, local reference.Local) (reference.Global, error) {
	if base := JetLocalRefOf(reference.New(base, local)); !base.IsEmpty() {
		return reference.NewSelf(base), nil
	}
	return reference.Global{}, newRefTypeErr(ErrUnexpectedJetRef, Jet, base, local)
}

func (typeDefJet) VerifyGlobalRef(base, local reference.Local) error {
	_, err := UnpackJetRef(reference.New(base, local))
	return err
}

func (typeDefJet) VerifyLocalRef(ref reference.Local) error {
	_, err := UnpackJetLocalRef(ref)
	return err
}

func (v typeDefJet) DetectSubType(base, local reference.Local) RefType {
	if base.IsEmpty() {
		if pulseZeroScope(local.GetHeader()) == pulse.Jet {
			return v.detectJetSubType(local, false)
		}
		return Invalid
	}

	switch {
	case pulseZeroScope(base.GetHeader()) != pulse.Jet:
	case base.IsZero():
		fallthrough
	case base == local:
		return v.detectJetSubType(local, false)
	case local.Pulse().IsTimePulse():
		if v.detectJetSubType(local, false) == Jet {
			return JetRecord
		}
	}
	return Invalid
}

func (typeDefJet) detectJetSubType(base reference.Local, mustBeDrop bool) RefType {
	switch pn, id := jetDropOfLocalRef(base, mustBeDrop); {
	case pn == invalidJetDropRef:
		return Invalid
	case pn == pulse.Unknown:
		return Jet
	case id.HasLength():
		return JetLeg
	default:
		return JetDrop
	}
}

var _ RefTypeDef = typeDefJetDrop{}
type typeDefJetDrop struct {}

func (typeDefJetDrop) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefJetDrop) Usage() Usage {
	return UseAsSelf|UseAsLocalValue
}

func (typeDefJetDrop) RefFrom(base, local reference.Local) (reference.Global, error) {
	id, err := UnpackJetDropRef(reference.New(base, local))
	if err == nil {
		return JetDropRef(id), nil
	}
	return reference.Global{}, err
}

func (typeDefJetDrop) VerifyGlobalRef(base, local reference.Local) error {
	_, err := UnpackJetDropRef(reference.New(base, local))
	return err
}

func (typeDefJetDrop) VerifyLocalRef(ref reference.Local) error {
	_, err := UnpackJetDropLocalRef(ref)
	return err
}

func (typeDefJetDrop) DetectSubType(_, _ reference.Local) RefType {
	panic(throw.Unsupported()) // it is a sub type
}

var _ RefTypeDef = typeDefJetLeg{}
type typeDefJetLeg struct {}

func (typeDefJetLeg) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefJetLeg) Usage() Usage {
	return UseAsSelf|UseAsLocalValue
}

func (typeDefJetLeg) RefFrom(base, local reference.Local) (reference.Global, error) {
	id, err := UnpackJetLegRef(reference.New(base, local))
	if err == nil {
		return JetLegRef(id), nil
	}
	return reference.Global{}, err
}

func (typeDefJetLeg) VerifyGlobalRef(base, local reference.Local) error {
	_, err := UnpackJetLegRef(reference.New(base, local))
	return err
}

func (typeDefJetLeg) VerifyLocalRef(ref reference.Local) error {
	_, err := UnpackJetLegLocalRef(ref)
	return err
}

func (typeDefJetLeg) DetectSubType(_, _ reference.Local) RefType {
	panic(throw.Unsupported()) // it is a sub type
}
