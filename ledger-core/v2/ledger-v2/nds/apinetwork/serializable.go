// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

type SerializationContext interface {
	PrepareHeader(*Header, pulse.Number) (pulse.Number, error)
	VerifyHeader(*Header, pulse.Number) error
	GetPayloadSigner() cryptkit.DataSigner
	GetPayloadEncrypter() cryptkit.Encrypter
}

type DeserializationContext interface {
	VerifyHeader(*Header, pulse.Number) error
	GetPayloadVerifier() cryptkit.DataSignatureVerifier
	GetPayloadDecrypter() cryptkit.Decrypter
	GetPayloadFactory() DeserializationFactory
}

type Serializer interface {
	SerializeTo(SerializationContext, io.Writer) error
}

type Deserializer interface {
	DeserializeFrom(DeserializationContext, io.Reader) error
}

type SizeAwareSerializer interface {
	ByteSize() uint
	SerializeTo(ctx SerializationContext, writer *iokit.LimitedWriter) error
}

type SizeAwareDeserializer interface {
	DeserializeFrom(DeserializationFactory, *iokit.LimitedReader) error
}

type Serializable interface {
	SizeAwareSerializer
	SizeAwareDeserializer
}

type PayloadCompleteness uint8

const (
	CompletePayload PayloadCompleteness = iota
	BodyPayload
	HeadPayload
)

type DeserializationFactory interface {
	DeserializePayloadFrom(PayloadCompleteness, *iokit.LimitedReader) (Serializable, error)
}
