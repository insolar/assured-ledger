// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type LineKey reference.Global

type LineSharedData struct {
	lineRef reference.Global
	limiter smsync.SemaphoreLink
}

func (p *LineSharedData) LineRef() reference.Global {
	return p.lineRef
}

func (p *LineSharedData) GetLimiter() smachine.SyncLink {

}

func (p *LineSharedData) TryApplyRecordSet(set inspectsvc.InspectedRecordSet) (*Future, interface{}) {

}

func (p *LineSharedData) RequestMissings(missings interface{}, wakeup smachine.BargeInHolder) {

}

func (p *LineSharedData) IsValid() bool {

}
