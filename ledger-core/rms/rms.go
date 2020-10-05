// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
)

type Any = rmsbox.Any
type AnyLazy = rmsbox.AnyLazy
type AnyLazyCopy = rmsbox.AnyLazyCopy

type AnyRecord = rmsbox.AnyRecord
type AnyRecordLazy = rmsbox.AnyRecordLazy
type AnyRecordLazyCopy = rmsbox.AnyRecordLazyCopy

type Binary = rmsbox.Binary
type RawBinary = rmsbox.RawBinary

type PulseNumber = pulse.Number
type PrimaryRole = member.PrimaryRole
type SpecialRole = member.SpecialRole
type ShortNodeID = node.ShortNodeID
type StorageLocator = ledger.StorageLocator
type ExtensionID = ledger.ExtensionID
type CatalogOrdinal = ledger.Ordinal
type DropOrdinal = ledger.DropOrdinal

type RecordVisitor = rmsbox.RecordVisitor
type BasicRecord  = rmsbox.BasicRecord
type MessageVisitor = rmsbox.MessageVisitor
type BasicMessage = rmsbox.BasicMessage
type Reference = rmsbox.Reference

type RecordBody = rmsbox.RecordBody
type RecordBodyDigests = rmsbox.RecordBodyDigests

type JetID = jet.ID
type JetDropID = jet.DropID
type JetLegID = jet.LegID

func RegisterRecordType(id uint64, special string, t BasicRecord) {
	rmsreg.GetRegistry().PutSpecial(id, special, reflect.TypeOf(t))
}

func RegisterMessageType(id uint64, special string, t proto.Message) {
	rmsreg.GetRegistry().PutSpecial(id, special, reflect.TypeOf(t))
}

func NewReference(v reference.Holder) Reference {
	return rmsbox.NewReference(v)
}

func NewReferenceLazy(v rmsbox.ReferenceProvider) Reference {
	return rmsbox.NewReferenceLazy(v)
}

func NewReferenceLocal(v reference.LocalHolder) Reference {
	return rmsbox.NewReferenceLocal(v)
}

func NewBytes(b []byte) Binary {
	return rmsbox.NewRawBytes(b).AsBinary()
}
