// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type Serializable interface {
	proto.ProtoSizer
	MarshalTo([]byte) (int, error)
	Unmarshal(b []byte) error
}

type DigestProvider interface {
	GetDigest() cryptkit.Digest
	MustDigest() cryptkit.Digest
}

type SerializableWithDigest struct {
	delegate Serializable
}

type SerializableWithReference interface {
	Serializable
}
