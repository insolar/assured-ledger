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

func RecordPayloadRef(id jet.ID, recordRef reference.LocalHolder, offset, length, extID uint32) reference.Global {
	local := recordRef.GetLocal()
	if !pulseZeroScope(local.GetHeader()).IsTimePulse() {
		panic(throw.IllegalValue())
	}

	var data reference.LocalHash
	EncodeJetData(&data, 0, jet.NoLengthExactID(id))
	EncodeJetPayloadPosition(&data, offset, length, extID)

	return reference.New(reference.NewLocal(pulse.Jet, 0, data), local)
}

func RecordPayloadRefByDrop(id jet.DropID, recordRef reference.LocalHolder, offset, length, extID uint32) reference.Global {
	pn := id.CreatedAt()
	if !pn.IsTimePulse() {
		panic(throw.IllegalValue())
	}

	local := recordRef.GetLocal()
	if pn != local.Pulse() {
		panic(throw.IllegalValue())
	}

	var data reference.LocalHash
	EncodeJetData(&data, 0, jet.NoLengthExactID(id.ID()))
	EncodeJetPayloadPosition(&data, offset, length, extID)

	return reference.New(reference.NewLocal(pulse.Jet, 0, data), local)
}

func UnpackRecordPayloadRef(ref reference.Holder) (id jet.ID, local reference.Local, offset, length, extID uint32, err error) {
	base := ref.GetBase()
	local = ref.GetLocal()

	id, err = PartialUnpackJetLocalRef(base)
	switch {
	case err == nil:
		offset, length, extID, err = DecodeJetPayloadPosition(base.IdentityHash(), true)
	case !pulseZeroScope(local.GetHeader()).IsTimePulse():
		err = throw.W(ErrIllegalRefValue,"invalid local")
	}

	if err != nil {
		err = newRefTypeErr(err, RecordPayload, base, local)
	}
	return
}

func DecodeJetPayloadPosition(b reference.LocalHash, checkTailingZeros bool) (offset, length, extID uint32, err error) {
	encoder := binary.LittleEndian
	length = encoder.Uint32(b[jetDataLength:jetDataLength+4])
	if length == 0 {
		err = throw.W(ErrIllegalRefValue,"empty jet payload")
		return
	}
	offset = encoder.Uint32(b[jetDataLength+4:jetDataLength+8])
	extID = encoder.Uint32(b[jetDataLength+8:jetDataLength+12])

	if checkTailingZeros {
		for _, bi := range b[jetDataLength+12:] {
			if bi != 0 {
				err = throw.W(ErrIllegalRefValue,"unexpected data in payload ref")
				return
			}
		}
	}
	return
}

func EncodeJetPayloadPosition(b *reference.LocalHash, offset, length, extID uint32) {
	if length == 0 {
		panic(throw.IllegalValue())
	}

	encoder := binary.LittleEndian
	encoder.PutUint32(b[jetDataLength:jetDataLength+4], length)
	encoder.PutUint32(b[jetDataLength+4:jetDataLength+8], offset)
	encoder.PutUint32(b[jetDataLength+8:jetDataLength+12], extID)
}

/*****************************************************/

var _ RefTypeDef = typeDefRecPayload{}
type typeDefRecPayload struct {}

func (typeDefRecPayload) CanBeDerivedWith(pulse.Number, reference.Local) bool {
	return false
}

func (typeDefRecPayload) Usage() Usage {
	return UseAsBase
}

func (v typeDefRecPayload) RefFrom(base, local reference.Local) (reference.Global, error) {
	if err := v.VerifyGlobalRef(base, local); err != nil {
		return reference.Global{}, err
	}
	return reference.New(base, local), nil
}

func (typeDefRecPayload) VerifyGlobalRef(base, local reference.Local) error {
	_, _, _, _, _, err := UnpackRecordPayloadRef(reference.New(base, local))
	return err
}

func (typeDefRecPayload) VerifyLocalRef(ref reference.Local) error {
	panic(throw.Unsupported())
}

func (typeDefRecPayload) DetectSubType(_, _ reference.Local) RefType {
	panic(throw.Unsupported()) // it is a sub type
}

