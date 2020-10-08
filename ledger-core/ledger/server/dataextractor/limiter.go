// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Limiter interface {
	CanRead() bool
	CanReadRecord() bool
	CanReadPayloads() bool
	CanReadExtension(id ledger.ExtensionID) bool

	First(reference.Holder)
	Next(consumedSize int, nextRef reference.Holder)
}

type Limits struct {
	Size uint64

	Entries  uint32
	Records  uint32
	Payloads uint32
	Extensions uint32

	StopRef reference.Holder
	IncludeStop bool
}

func NewLimiter(limits Limits) Limiter {
	return &limiter{
		remainedSize: limits.Size,
		limitEnt:     limits.Entries,
		limitRec:     limits.Records,
		limitPld:     limits.Payloads,
		limitExt:     limits.Extensions,
		stopRef:      reference.CopyLocal(limits.StopRef),
		includeStop:  limits.IncludeStop,
	}
}


var _ Limiter = &limiter{}
type limiter struct {
	remainedSize uint64

	count uint32

	limitEnt uint32
	limitRec uint32
	limitPld uint32
	limitExt uint32

	stopRef reference.Local
	includeStop bool
}

func (p *limiter) CanRead() bool {
	return p.count > 0 && p.count <= p.limitEnt
}

func (p *limiter) CanReadRecord() bool {
	return p.CanRead() && p.count <= p.limitRec
}

func (p *limiter) CanReadPayloads() bool {
	return p.CanRead() && p.count <= p.limitPld
}

func (p *limiter) CanReadExtension(ledger.ExtensionID) bool {
	return p.CanRead() && p.count <= p.limitExt
}

func (p *limiter) checkStop(ref reference.Holder) {
	switch {
	case p.stopRef.IsEmpty():
	case p.stopRef != ref.GetLocal():
	case !p.CanRead():
	case p.includeStop:
		p.limitEnt = p.count
	default:
		p.limitEnt = p.count - 1
	}
}

func (p *limiter) First(ref reference.Holder) {
	if p.count != 0 {
		panic(throw.IllegalState())
	}
	p.count = 1
	// NB! Size limit is NOT applied for the first record
	p.checkStop(ref)
}

func (p *limiter) Next(consumedSize int, nextRef reference.Holder) {
	if p.count == 0 {
		panic(throw.IllegalState())
	}
	p.count++

	switch {
	case consumedSize <= 0:
		panic(throw.IllegalValue())
	case uint64(consumedSize) >= p.remainedSize:
		p.remainedSize = 0
		p.limitEnt = 0
	default:
		p.remainedSize -= uint64(consumedSize)
	}
	p.checkStop(nextRef)
}
