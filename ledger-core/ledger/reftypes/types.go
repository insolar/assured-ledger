// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type RefTypeDef interface {
	Usage() Usage
	VerifyGlobalRef(base, local reference.Local) error
	VerifyLocalRef(reference.Local) error
	DetectSubType(base, local reference.Local) RefType
}

const (
	UseAsBase Usage = 1<<iota
	UseAsSelf
	UseAsLocal
	UseAsLocalValue
)

// method is used instead of an array for compiler optimization
func typeDefinition(t RefType) RefTypeDef {
	switch t {
	case Node:
		return tDefNode
	case NodeContract:
		return tDefNodeContract
	case APICall:
		return tDefAPICall
	case Jet:
		return tDefJet
	case JetDrop:
		return tDefJetDrop
	case JetLeg:
		return tDefJetLeg
	case RecordPayload:
		return tDefRecPayload
	case BuiltinContract:
		return tDefBuiltinContract
	default:
		return nil
	}
}

var (
	tDefAPICall = typeDefAPICall{}
	tDefNode = typeDefNode{}
	tDefNodeContract = typeDefNodeContract{}
	tDefJet = typeDefJet{}
	tDefJetDrop = typeDefJetDrop{}
	tDefJetLeg = typeDefJetLeg{}
	tDefRecPayload = typeDefRecPayload{}
	tDefBuiltinContract = typeDefBuiltinContract{}
)

// APISession128: {usage: UseAsSelf},
// APISession384: {usage: UseAsBase },
//
// APIEndpoint: {usage: UseAsSelf| UseAsLocalValue},
