// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"hash/crc32"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// GenerateUintShortID generate short ID for node without checking collisions
func GenerateUintShortID(ref reference.Global) uint32 {
	return crc32.ChecksumIEEE(ref.AsBytes())
}
