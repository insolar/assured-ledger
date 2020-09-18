// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

type RequestResult struct {
	SideEffectType     Type             // every
	RawResult          []byte           // every
	RawObjectReference reference.Global // every

	ObjectImage reference.Global // amend + activate
	Memory      []byte           // amend + activate
}

func New(result []byte, objectRef reference.Global) *RequestResult {
	return &RequestResult{
		SideEffectType:     SideEffectNone,
		RawResult:          result,
		RawObjectReference: objectRef,
	}
}

func (s *RequestResult) Result() []byte {
	return s.RawResult
}

func (s *RequestResult) Activate() (reference.Global, []byte) {
	return s.ObjectImage, s.Memory
}

func (s *RequestResult) Amend() (reference.Global, []byte) {
	return s.ObjectImage, s.Memory
}

func (s *RequestResult) Deactivate() (reference.Global, []byte) {
	return s.ObjectImage, s.Memory
}

func (s *RequestResult) SetActivate(image reference.Global, memory []byte) {
	s.SideEffectType = SideEffectActivate

	s.ObjectImage = image
	s.Memory = memory
}

func (s *RequestResult) SetAmend(object descriptor.Object, memory []byte) {
	s.SideEffectType = SideEffectAmend
	s.Memory = memory

	class, _ := object.Class()
	s.ObjectImage = class
}

func (s *RequestResult) SetDeactivate(object descriptor.Object) {
	s.SideEffectType = SideEffectDeactivate
	class, _ := object.Class()
	s.ObjectImage = class
}

func (s RequestResult) Type() Type {
	return s.SideEffectType
}

func (s *RequestResult) ObjectReference() reference.Global {
	return s.RawObjectReference
}
