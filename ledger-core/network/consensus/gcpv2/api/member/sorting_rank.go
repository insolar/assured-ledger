package member

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
)

type SortingRank struct {
	nodeID    node.ShortNodeID
	powerRole uint16
}

func NewSortingRank(nodeID node.ShortNodeID, role PrimaryRole, pw Power, mode OpMode) SortingRank {
	return SortingRank{nodeID, SortingPowerRole(role, pw, mode)}
}

func (v SortingRank) GetNodeID() node.ShortNodeID {
	return v.nodeID
}

func (v SortingRank) IsWorking() bool {
	return v.powerRole != 0
}

func (v SortingRank) GetWorkingRole() PrimaryRole {
	return PrimaryRole(v.powerRole >> 8)
}

func (v SortingRank) GetPower() Power {
	return Power(v.powerRole)
}

// NB! Sorting is REVERSED
func (v SortingRank) Less(o SortingRank) bool {
	if o.powerRole < v.powerRole {
		return true
	}
	if o.powerRole > v.powerRole {
		return false
	}
	return o.nodeID < v.nodeID
}

// NB! Sorting is REVERSED
func LessByID(vNodeID, oNodeID node.ShortNodeID) bool {
	return oNodeID < vNodeID
}

func SortingPowerRole(role PrimaryRole, pw Power, mode OpMode) uint16 {
	if role == 0 {
		panic("illegal value")
	}
	if pw == 0 || mode.IsPowerless() {
		return 0
	}
	return uint16(role)<<8 | uint16(pw)
}
