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
	RootRef() reference.Holder
	CurrentRef() reference.Holder
	CurrentLoc() ledger.StorageLocator

	Next() bool
}

type forwardIterator struct {
	// anti-PrevRef
}

type backwardIterator struct {
	// PrefRef
}
