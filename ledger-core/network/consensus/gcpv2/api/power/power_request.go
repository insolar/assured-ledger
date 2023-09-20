package power

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/capacity"
)

type Request int16

const EmptyRequest Request = 0

func NewRequestByLevel(v capacity.Level) Request {
	return -Request(v) - 1
}

func NewRequest(v member.Power) Request {
	return Request(v) + 1
}

func (v Request) AsCapacityLevel() (bool, capacity.Level) {
	return v < 0, capacity.Level(-(v + 1))
}

func (v Request) AsMemberPower() (bool, member.Power) {
	return v > 0, member.Power(v - 1)
}

func (v Request) IsEmpty() bool {
	return v == EmptyRequest
}

func (v Request) Update(pw *member.Power, set member.PowerSet) bool {
	prev := *pw
	if ok, cl := v.AsCapacityLevel(); ok {
		npw := set.ForLevel(cl)
		*pw = npw
		return prev != npw
	}
	if ok, npw := v.AsMemberPower(); ok {
		npw = set.FindNearestValid(npw)
		*pw = npw
		return prev != npw
	}
	return false
}
