// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type ByteString = longbits.ByteString

type GoGoMarshaller interface {
	proto.ProtoSizer
	MarshalTo(dAtA []byte) (int, error)
	proto.Unmarshaler
}

type ProtoMessage interface {
	proto.Marshaler
	proto.Unmarshaler
	//Reset()
}

type GoGoMessage interface {
	GoGoMarshaller
	proto.Marshaler
	//Reset()
}

type PolymorphMessage interface {
	GoGoMarshaller
	// deprecated
	InitPolymorphField(setup bool) bool
}
