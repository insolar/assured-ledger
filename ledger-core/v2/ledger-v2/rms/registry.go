// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"sync"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PolymorphHack interface {
	InitPolymorphField(setup bool) bool
}

type PolymorphFactoryFunc func() ProtoMessage

var polymorphTypes sync.Map

func RegisterPolymorph(id Polymorph, factoryFn PolymorphFactoryFunc) {
	switch {
	case factoryFn == nil:
		panic(throw.IllegalValue())
	case id <= PolymorphInplace:
		panic(throw.IllegalValue())
	}

	if _, loaded := polymorphTypes.LoadOrStore(id, factoryFn); loaded {
		panic(throw.IllegalState())
	}
}

func getPolymorph(id Polymorph) ProtoMessage {
	if id > PolymorphInplace {
		if v, ok := polymorphTypes.Load(id); ok {
			if fn, ok := v.(PolymorphFactoryFunc); ok {
				if msg := fn(); msg != nil {
					return msg
				}
				panic(throw.Impossible())
			}
		}
	}
	return nil
}

const PolymorphFieldId = 16

//var fieldPolymorph = protokit.WireVarint.Tag(PolymorphFieldId)

func UnmarshalAnyPolymorph(b []byte) (interface{}, error) {
	if msg, err := PeekPolymorph(b); err != nil {
		return nil, err
	} else if err = msg.Unmarshal(b); err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}

func PeekPolymorph(b []byte) (ProtoMessage, error) {
	switch n := len(b); {
	case n < 3:
		if n == 0 {
			return nil, nil
		}
	case b[0] == 0x80 && b[1] == 0x01:
		switch id, nn := proto.DecodeVarint(b[2:]); {
		case nn == 0:
		case id > uint64(^Polymorph(0)):
		default:
			return getPolymorph(Polymorph(id)), nil
		}
	}
	return nil, throw.IllegalValue()
}

func UnmarshalPolymorph(id Polymorph, b []byte) (interface{}, error) {
	switch {
	case id > PolymorphInplace:
		if msg := getPolymorph(id); msg != nil {
			if err := msg.Unmarshal(b); err != nil {
				return nil, err
			}
			return msg, nil
		}
	case id == PolymorphNone:
		return nil, throw.IllegalValue()
	}
	return nil, throw.IllegalValue()
}

func NewPolymorph(id Polymorph) interface{} {
	if obj := getPolymorph(id); obj != nil {
		return obj
	}
	return nil
}
