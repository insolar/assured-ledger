// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
)

var _ rmsreg.GoGoSerializableWithText = &Binary{}
var _ longbits.FixedReader = &Binary{}

type Binary struct {
	rawBinary
}
type rawBinary = RawBinary

func (p *Binary) ProtoSize() int {
	return protokit.BinaryProtoSize(p.rawBinary.protoSize())
}

func (p *Binary) MarshalTo(b []byte) (int, error) {
	return protokit.BinaryMarshalTo(b, p.rawBinary.marshalTo)
}

func (p *Binary) MarshalToSizedBuffer(b []byte) (int, error) {
	return protokit.BinaryMarshalToSizedBuffer(b, p.rawBinary.marshalToSizedBuffer)
}

func (p *Binary) Unmarshal(b []byte) error {
	return protokit.BinaryUnmarshal(b, p.rawBinary.unmarshal)
}

func (p *Binary) Equal(o interface{}) bool {
	switch ov := o.(type) {
	case nil:
		return p == nil || p.rawBinary.IsEmpty()
	case *Binary:
		switch {
		case p == ov:
			return true
		case ov == nil:
			return p == nil || p.rawBinary.IsEmpty()
		case p == nil:
			return ov.rawBinary.IsEmpty()
		}
		return p.rawBinary._equal(&ov.rawBinary)
	case longbits.FixedReader:
		return p.rawBinary._equalReader(ov)
	}
	return false
}
