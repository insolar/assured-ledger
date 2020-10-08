// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Limits struct {
	Size uint64

	Entries  uint32
	Records  uint32
	Payloads uint32
	Extensions uint32

	StopRef reference.Holder
	ExcludeStop bool
	ExcludeStart bool
}

type limiter struct {
	limits Limits

	limitCount uint32
	count      uint32
	accumSize  uint64
}

func (p *limiter) CanRead() bool {
	return p.count <= p.limitCount
}

func (p *limiter) CanReadRecord() bool {
	return p.CanRead() && p.count <= p.limits.Records
}

func (p *limiter) CanReadPayloads() bool {
	return p.CanRead() && p.count <= p.limits.Payloads
}

func (p *limiter) CanReadExtension(ledger.ExtensionID) bool {
	return p.CanRead() && p.count <= p.limits.Extensions
}

func (p *limiter) Next(consumedSize int, nextRef reference.Holder) {
	switch {
	case consumedSize < 0:
		panic(throw.IllegalValue())
	case p.count == 0:
		// NB! Size limit is NOT applied to the first record
		p.limitCount = p.limits.Entries
		p.accumSize = 0
	case !p.CanRead():
		return
	case consumedSize == 0:
	case p.accumSize == math.MaxUint64:
	default:
		sz := p.accumSize + uint64(consumedSize)
		if sz < p.accumSize {
			// overflow
			p.accumSize = math.MaxUint64
		} else {
			p.accumSize = sz
		}

		if p.accumSize > p.limits.Size {
			p.limitCount = p.count
		}
	}

	switch {
	case nextRef == nil || reference.IsEmpty(p.limits.StopRef):
	case p.limits.StopRef.GetLocal() != nextRef.GetLocal():
	case p.limits.ExcludeStop:
		p.limitCount = p.count
	default:
		p.limitCount = p.count + 1
	}

	p.count++
}
