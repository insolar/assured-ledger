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
