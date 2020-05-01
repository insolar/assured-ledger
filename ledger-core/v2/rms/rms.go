// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type ByteString = longbits.ByteString
type RecordBody = BlobBody
type RecordExtension = BlobBody
type PulseNumber = pulse.Number

type ContextMessage interface {
	SetupContext(ctx Context) error
}

type FieldMapper interface {
	ContextMessage
	GetFieldMap() insproto.FieldMap
}

type Context interface {
	RecordBodyHash(*RecordBody) error
	RecordExtensionHash(*[]RecordExtension) error
	Record(FieldMapper) error
}
