// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
)

type Output interface {
	BeginEntry(entry *catalog.Entry, fullExcerpt bool) error
	AddBody(readbundle.Slice) error
	AddPayload(readbundle.Slice) error
	AddExtension(ledger.ExtensionID, readbundle.Slice) error
	EndEntry() (consumedSize int, err error)
}
