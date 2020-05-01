// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type RequestResult struct {
	SideEffectType     insolar.RequestResultType // every
	RawResult          []byte                    // every
	RawObjectReference reference.Global          // every

	ParentReference reference.Global // activate
	ObjectImage     reference.Global // amend + activate
	ObjectStateID   reference.Local  // amend + deactivate
	Memory          []byte           // amend + activate
}

func New(result []byte, objectRef reference.Global) *RequestResult {
	return &RequestResult{
		SideEffectType:     insolar.RequestSideEffectNone,
		RawResult:          result,
		RawObjectReference: objectRef,
	}
}

func (s *RequestResult) Result() []byte {
	return s.RawResult
}

func (s *RequestResult) Activate() (reference.Global, reference.Global, []byte) {
	return s.ParentReference, s.ObjectImage, s.Memory
}

func (s *RequestResult) Amend() (reference.Local, reference.Global, []byte) {
	return s.ObjectStateID, s.ObjectImage, s.Memory
}

func (s *RequestResult) Deactivate() reference.Local {
	return s.ObjectStateID
}

func (s *RequestResult) SetActivate(parent, image reference.Global, memory []byte) {
	s.SideEffectType = insolar.RequestSideEffectActivate

	s.ParentReference = parent
	s.ObjectImage = image
	s.Memory = memory
}

func (s *RequestResult) SetAmend(object descriptor.ObjectDescriptor, memory []byte) {
	s.SideEffectType = insolar.RequestSideEffectAmend
	s.Memory = memory
	s.ObjectStateID = *object.StateID()

	prototype, _ := object.Prototype()
	s.ObjectImage = *prototype
}

func (s *RequestResult) SetDeactivate(object descriptor.ObjectDescriptor) {
	s.SideEffectType = insolar.RequestSideEffectDeactivate
	s.ObjectStateID = *object.StateID()
}

func (s RequestResult) Type() insolar.RequestResultType {
	return s.SideEffectType
}

func (s *RequestResult) ObjectReference() reference.Global {
	return s.RawObjectReference
}
