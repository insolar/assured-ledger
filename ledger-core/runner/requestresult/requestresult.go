// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RequestResult struct {
	effects Type
	result  []byte
	class   reference.Global
	memory  []byte
}

func (s RequestResult) Effects() Type {
	return s.effects
}

func (s RequestResult) HasEffects() bool {
	return s.effects != 0
}

func (s RequestResult) HasMemoryAmend() bool {
	return s.effects&SideEffectAmend != 0
}

func (s RequestResult) IsActivation() bool {
	return s.effects&SideEffectActivate != 0
}

func (s RequestResult) IsDeactivation() bool {
	return s.effects&SideEffectDeactivate != 0
}

func (s RequestResult) CallResult() []byte {
	return s.result
}

func (s RequestResult) Memory() []byte {
	return s.memory
}

func (s RequestResult) Class() reference.Global {
	if s.class.IsEmpty() {
		panic(throw.IllegalState())
	}
	return s.class
}

type ResultBuilder struct {
	res RequestResult
}

func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{}
}

func (s *ResultBuilder) Class(class reference.Global) *ResultBuilder {
	s.res.class = class
	return s
}

func (s *ResultBuilder) CallResult(res []byte) *ResultBuilder {
	s.res.result = res
	return s
}

func (s *ResultBuilder) Memory(memory []byte) *ResultBuilder {
	if memory == nil || len(memory) == 0 {
		panic(throw.IllegalValue())
	}

	s.res.effects |= SideEffectAmend
	s.res.memory = memory
	return s
}

func (s *ResultBuilder) Activate(memory []byte) *ResultBuilder {
	if memory == nil || len(memory) == 0 {
		panic(throw.IllegalValue())
	}

	s.res.effects |= SideEffectActivate | SideEffectAmend
	s.res.memory = memory
	return s
}

func (s *ResultBuilder) Deactivate() *ResultBuilder {
	s.res.effects |= SideEffectDeactivate
	return s
}

func (s *ResultBuilder) Result() RequestResult {
	if s.res.result == nil {
		panic(throw.IllegalState())
	}
	return s.res
}
