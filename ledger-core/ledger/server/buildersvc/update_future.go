// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type Future struct {

}

func (p *Future) GetReadySync() smachine.SyncLink {

}

func (p *Future) IsCommitted() bool {

}
