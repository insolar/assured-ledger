// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsreg

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewCachedSerializable(s Serializable, fn func([]byte)) CachedSerializable {
	return CachedSerializable{source: s, readyFn: fn}
}

var _ Serializable = &CachedSerializable{}

type CachedSerializable struct {
	cachedData []byte
	source     Serializable
	err        error
	readyFn    func([]byte)
}

func (p *CachedSerializable) ProtoSize() int {
	switch {
	case p.cachedData != nil:
		return len(p.cachedData)
	case p.err != nil:
		return 1
	case p.source == nil:
		return 0
	}

	p.cachedData, p.err = p.source.Marshal()
	p.source = nil
	if p.err != nil {
		p.cachedData = nil
		return 1
	}
	if p.readyFn != nil {
		p.readyFn(p.cachedData)
	}

	return len(p.cachedData)
}

func (p *CachedSerializable) Marshal() ([]byte, error) {
	switch {
	case p.err != nil:
		return nil, p.err
	case p.cachedData != nil:
		return append([]byte(nil), p.cachedData...), nil
	case p.source == nil:
		return nil, nil
	default:
		panic(throw.IllegalState())
	}
}

func (p *CachedSerializable) MarshalTo(b []byte) (int, error) {
	switch {
	case p.err != nil:
		return 0, p.err
	case p.cachedData != nil:
		n := len(p.cachedData)
		if len(b) < n {
			return 0, io.ErrShortBuffer
		}
		return copy(b, p.cachedData), nil
	case p.source == nil:
		return 0, nil
	default:
		panic(throw.IllegalState())
	}
}

func (p *CachedSerializable) Unmarshal(b []byte) error {
	switch {
	case p.source != nil:
		return p.source.Unmarshal(b)
	case p.err != nil:
		return p.err
	// case p.cachedData != nil:
	// 	panic(throw.IllegalState())
	default:
		if p.readyFn != nil {
			p.readyFn(p.cachedData)
		} else {
			p.cachedData = b
		}
		return nil
	}
}
