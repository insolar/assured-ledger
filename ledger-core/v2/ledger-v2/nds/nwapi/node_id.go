// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"math"
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type HostId uint64

func (v HostId) IsAbsent() bool { return v == AbsentHostId }

func (v HostId) IsNodeId() bool { return v > 0 && v <= maxShortNodeId }

func (v HostId) AsNodeId() ShortNodeID {
	if v.IsNodeId() {
		return ShortNodeID(v)
	}
	panic(throw.IllegalState())
}

func (v HostId) String() string {
	if v <= maxShortNodeId {
		return strconv.FormatUint(uint64(v), 10)
	}
	return strconv.FormatUint(uint64(v&maxShortNodeId), 10) +
		"@[" + strconv.FormatUint(uint64(v>>(ShortNodeIDByteSize*8)), 10) + "]"
}

const (
	AbsentHostId   HostId = 0
	HostIdByteSize        = 8
)

// ShortNodeID is the shortened ID of node that is unique inside the globe
type ShortNodeID uint32

const (
	AbsentShortNodeID   ShortNodeID = 0
	ShortNodeIDByteSize             = 4
	maxShortNodeId                  = math.MaxUint32
)

func (v ShortNodeID) IsAbsent() bool { return v == AbsentShortNodeID }

// deprecated
func (v ShortNodeID) Equal(other ShortNodeID) bool { return v == other }

func (v ShortNodeID) String() string {
	return strconv.FormatUint(uint64(v), 10)
}
