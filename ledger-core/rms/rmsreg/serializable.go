package rmsreg

import (
	"encoding"

	"github.com/gogo/protobuf/proto"
)

type unmarshaler interface {
	Unmarshal([]byte) error
}

// UnknownCallbackFunc handles unknown field during unmarshalling.
// This func can return:
// - positive value to indicate that size of the field
// - zero to use default skip behavior
// - negative value to ignore all further fields
type UnknownCallbackFunc = func([]byte) (int, error)

type unmarshalerWithUnknownCallback interface {
	UnmarshalWithUnknownCallback([]byte, UnknownCallbackFunc) (int, error)
}

type Serializable interface {
	proto.ProtoSizer
	unmarshaler
	Marshal() ([]byte, error)
}

type MarshalerTo interface {
	proto.ProtoSizer
	MarshalTo([]byte) (int, error)
}

type GoGoMarshaler interface {
	proto.ProtoSizer
	MarshalTo([]byte) (int, error)
	MarshalToSizedBuffer([]byte) (int, error)
}

type GoGoSerializable interface {
	GoGoMarshaler
	unmarshaler
}

type GoGoSerializableWithText interface {
	GoGoSerializable
	encoding.TextMarshaler
}
