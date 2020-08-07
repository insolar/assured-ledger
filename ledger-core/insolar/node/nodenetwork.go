// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"hash/crc32"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

const (
	ShortNodeIDSize = 4
)

type ShortNodeID = nwapi.ShortNodeID

const AbsentShortNodeID = nwapi.AbsentShortNodeID

// GenerateShortID generate short ID for node without checking collisions
func GenerateShortID(ref reference.Holder) ShortNodeID {
	return ShortNodeID(crc32.ChecksumIEEE(reference.AsBytes(ref)))
}
