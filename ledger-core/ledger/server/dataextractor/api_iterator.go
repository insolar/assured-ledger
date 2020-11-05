// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Iterator interface {
	Direction() Direction
	CurrentEntry() ledger.DirectoryIndex
	ExtraEntries() []ledger.DirectoryIndex

	// Next retrieves a next entry in a sequence. Arg (prevRef) should be provided irrelevant of direction
	Next(prevRef reference.Holder) (bool, error)
}

