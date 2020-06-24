// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"encoding"

	"github.com/gogo/protobuf/proto"
)

type unmarshaler interface {
	Unmarshal([]byte) error
}

type UnknownCallbackFunc = func([]byte) (int, error)
type unmarshalerWithUnknownCallback interface {
	UnmarshalWithUnknownCallback([]byte, UnknownCallbackFunc) error
}

type Serializable interface {
	proto.ProtoSizer
	unmarshaler
	Marshal() ([]byte, error)
}

type goGoMarshaler interface {
	proto.ProtoSizer
	MarshalTo([]byte) (int, error)
	MarshalToSizedBuffer([]byte) (int, error)
}

type GoGoSerializable interface {
	goGoMarshaler
	unmarshaler
	encoding.TextMarshaler
}

type GoGoSerializableWithText interface {
	GoGoSerializable
	encoding.TextMarshaler
}
