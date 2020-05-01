// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type RequestResult struct {
	RawResult          []byte           // every
	RawObjectReference reference.Global // every

	ParentReference reference.Global // activate
	ObjectImage     reference.Global // amend + activate
	ObjectStateID   reference.Local  // amend + deactivate
	Memory          []byte           // amend + activate
}

func New(result []byte, objectRef reference.Global) *RequestResult {
	return &RequestResult{
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

func (s *RequestResult) ObjectReference() reference.Global {
	return s.RawObjectReference
}
