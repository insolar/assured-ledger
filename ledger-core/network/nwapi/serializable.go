// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
)

type SerializationContext interface {
	SerializationContext()
	// GetPayloadSigner() cryptkit.DataSigner
	// GetPayloadEncrypter() cryptkit.Encrypter
}

type DeserializationContext interface {
	// GetPayloadVerifier() cryptkit.DataSignatureVerifier
	// GetPayloadDecrypter() cryptkit.Decrypter
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
	SerializeTo(SerializationContext, *iokit.LimitedWriter) error
}

type SizeAwareDeserializer interface {
	DeserializeFrom(DeserializationContext, *iokit.LimitedReader) error
}

type Serializable interface {
	SizeAwareSerializer
	SizeAwareDeserializer
}

type PayloadCompleteness bool

const (
	CompletePayload PayloadCompleteness = true
	PartialPayload  PayloadCompleteness = false
	HeadOnlyPayload                     = CompletePayload
)

type DeserializationFactory interface {
	DeserializePayloadFrom(DeserializationContext, PayloadCompleteness, *iokit.LimitedReader) (Serializable, error)
}
