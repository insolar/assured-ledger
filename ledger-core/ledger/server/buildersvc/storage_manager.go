// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newStorageWriter(pn pulse.Number) *storageWriter {
	return &storageWriter{
		writeChan: make(chan writeBundle, 32),
		oobChan: make(chan oobEvent, 2),
	}
}

type storageWriter struct {
	writeChan chan writeBundle // close on pulse change
	oobChan   chan oobEvent    // close on pulse change
}

type oobEvent struct {
	// suspend & get NSH
	// resume
}

type writeBundle struct {
	jetDrop jet.DropID
	bundle  lineage.BundleResolver
	future  *Future
}
