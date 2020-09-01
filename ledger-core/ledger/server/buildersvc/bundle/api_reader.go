// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bundle

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// DirtyReader provides dirty read access to SnapshotWriter
type DirtyReader interface {
	GetDirectoryEntry(ledger.DirectoryIndex) (key reference.Holder, loc ledger.StorageLocator)
	GetEntryStorage(ledger.StorageLocator) []byte
	GetPayloadStorage(ledger.StorageLocator) []byte
}
