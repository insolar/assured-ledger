// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"math"
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// LocalUniqueID is a protocol-independent locally unique id of a peer. This id wll change after de-registration of a peer.
type LocalUniqueID uint64

func (v LocalUniqueID) IsAbsent() bool { return v == 0 }

func (v LocalUniqueID) String() string {
	return strconv.FormatUint(uint64(v), 10)
}

// HostID is a protocol-depended numerical id of a peer
type HostID uint64

func (v HostID) IsAbsent() bool { return v == 0 }

func (v HostID) IsNodeID() bool { return v > 0 && v <= maxShortNodeID }

func (v HostID) AsNodeID() ShortNodeID {
	if v.IsNodeID() {
		return ShortNodeID(v)
	}
	panic(throw.IllegalState())
}

func (v HostID) String() string {
	if v <= maxShortNodeID {
		return strconv.FormatUint(uint64(v), 10)
	}
	return strconv.FormatUint(uint64(v&maxShortNodeID), 10) +
		"@[" + strconv.FormatUint(uint64(v>>(ShortNodeIDByteSize*8)), 10) + "]"
}

const (
	HostIDByteSize   = 8
	LocalUIDByteSize = 8
)

// ShortNodeID is the shortened ID of a peer that belongs to a member of a globula.
type ShortNodeID uint32

const (
	AbsentShortNodeID   ShortNodeID = 0
	ShortNodeIDByteSize             = 4
	maxShortNodeID                  = math.MaxUint32
)

func (v ShortNodeID) IsAbsent() bool { return v == AbsentShortNodeID }

// deprecated - do not use
func (v ShortNodeID) Equal(other ShortNodeID) bool { return v == other }

func (v ShortNodeID) String() string {
	return strconv.FormatUint(uint64(v), 10)
}
