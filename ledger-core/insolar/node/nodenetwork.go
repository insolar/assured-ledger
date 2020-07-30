// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"hash/crc32"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

const (
	ShortNodeIDSize = 4
)

// ShortNodeID is the shortened ID of node that is unique inside the globe
type ShortNodeID uint32 // ZERO is RESERVED

const AbsentShortNodeID ShortNodeID = 0

func (v ShortNodeID) IsAbsent() bool { return v == AbsentShortNodeID }

func (v ShortNodeID) Equal(other ShortNodeID) bool { return v == other }

// GenerateShortID generate short ID for node without checking collisions
func GenerateShortID(ref reference.Holder) ShortNodeID {
	return ShortNodeID(crc32.ChecksumIEEE(reference.AsBytes(ref)))
}
