package dataextractor

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewLimiter(limits Limits) Limiter {
	return &limiter{limits: limits}
}

var _ Limiter = &limiter{}
type limiter struct {
	limits Limits

	limitCount uint32
	count      uint32
	accumSize  uint64
}

func (p *limiter) Clone() Limiter {
	return NewLimiter(p.limits)
}

func (p *limiter) CanRead() bool {
	return p.count <= p.limitCount
}

func (p *limiter) IsSkipped(recType uint32) bool {
	switch {
	case p.count == 1:
		return p.limits.ExcludeStart
	case p.limits.RecTypeFilter != nil:
		return p.limits.RecTypeFilter(recType)
	default:
		return false
	}
}

func (p *limiter) CanReadExcerpt() bool {
	return p.CanRead() && p.count <= p.limits.Excerpts
}

func (p *limiter) CanReadBody() bool {
	return p.CanRead() && p.count <= p.limits.Bodies
}

func (p *limiter) CanReadPayload() bool {
	return p.CanRead() && p.count <= p.limits.Payloads
}

func (p *limiter) CanReadExtensions() bool {
	return p.CanRead() && p.count <= p.limits.Extensions
}

func (p *limiter) CanReadExtension(id ledger.ExtensionID) bool {
	switch {
	case !p.CanReadExtensions():
		return false
	case p.limits.ExtTypeFilter != nil:
		return p.limits.ExtTypeFilter(id)
	default:
		return true
	}
}

func (p *limiter) Next(consumedSize int, nextRecRef reference.Holder) {
	switch {
	case consumedSize < 0:
		panic(throw.IllegalValue())
	case p.count == 0:
		// NB! Size limit is NOT applied to the first record
		p.limitCount = p.limits.TotalCount
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

		if p.accumSize > p.limits.TotalSize {
			p.limitCount = p.count
		}
	}

	switch {
	case nextRecRef == nil || reference.IsEmpty(p.limits.StopRef):
	case p.limits.StopRef.GetLocal() != nextRecRef.GetLocal():
	case p.limits.ExcludeStop:
		p.limitCount = p.count
	default:
		p.limitCount = p.count + 1
	}

	p.count++
}
