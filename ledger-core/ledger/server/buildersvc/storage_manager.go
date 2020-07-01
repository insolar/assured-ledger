// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
)

type storageManager struct {
	writePipe chan struct{}
	// pulsePrepare signal
	// pulseCancel signal
}

type writeBundle struct {
	jetDrop JetDropID
	bundle  lineage.BundleResolver
	future  *Future
}
