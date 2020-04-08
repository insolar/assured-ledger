// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type RequestResult struct {
	SideEffectType     insolar.RequestResultType // every
	RawResult          []byte                    // every
	RawObjectReference insolar.Reference         // every

	ParentReference insolar.Reference // activate
	ObjectImage     insolar.Reference // amend + activate
	ObjectStateID   insolar.ID        // amend + deactivate
	Memory          []byte            // amend + activate
}

func New(result []byte, objectRef insolar.Reference) *RequestResult {
	return &RequestResult{
		SideEffectType:     insolar.RequestSideEffectNone,
		RawResult:          result,
		RawObjectReference: objectRef,
	}
}

func (s *RequestResult) Result() []byte {
	return s.RawResult
}

func (s *RequestResult) Activate() (insolar.Reference, insolar.Reference, []byte) {
	return s.ParentReference, s.ObjectImage, s.Memory
}

func (s *RequestResult) Amend() (insolar.ID, insolar.Reference, []byte) {
	return s.ObjectStateID, s.ObjectImage, s.Memory
}

func (s *RequestResult) Deactivate() insolar.ID {
	return s.ObjectStateID
}

func (s *RequestResult) SetActivate(parent, image insolar.Reference, memory []byte) {
	s.SideEffectType = insolar.RequestSideEffectActivate

	s.ParentReference = parent
	s.ObjectImage = image
	s.Memory = memory
}

func (s *RequestResult) SetAmend(object insolar.ObjectDescriptor, memory []byte) {
	s.SideEffectType = insolar.RequestSideEffectAmend
	s.Memory = memory
	s.ObjectStateID = *object.StateID()

	prototype, _ := object.Prototype()
	s.ObjectImage = *prototype
}

func (s *RequestResult) SetDeactivate(object insolar.ObjectDescriptor) {
	s.SideEffectType = insolar.RequestSideEffectDeactivate
	s.ObjectStateID = *object.StateID()
}

func (s RequestResult) Type() insolar.RequestResultType {
	return s.SideEffectType
}

func (s *RequestResult) ObjectReference() insolar.Reference {
	return s.RawObjectReference
}
