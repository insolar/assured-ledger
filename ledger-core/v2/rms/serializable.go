// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type Serializable interface {
	proto.ProtoSizer
	MarshalTo([]byte) (int, error)
	Unmarshal([]byte) error
}

type GoGoSerializable interface {
	proto.ProtoSizer
	MarshalTo([]byte) (int, error)
	MarshalToSizedBuffer([]byte) (int, error)
	Unmarshal([]byte) error
}

type DigestProvider interface {
	GetDigest() cryptkit.Digest
	MustDigest() cryptkit.Digest
}

var _ Serializable = &SerializableWithReference{}

type SerializableWithReference struct {
	underlying Serializable
}

func (p *SerializableWithReference) ProtoSize() int {
	panic(throw.NotImplemented())
}

func (p *SerializableWithReference) MarshalTo(b []byte) (int, error) {
	panic(throw.NotImplemented())
}

func (p *SerializableWithReference) Unmarshal(b []byte) error {
	panic(throw.NotImplemented())
}

func (p *SerializableWithReference) MustDigest() cryptkit.Digest {
	panic(throw.NotImplemented())
}
