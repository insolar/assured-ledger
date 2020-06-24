// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type StreamDropKey pulse.Number

type StreamDropCataloger interface {
	GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) StreamDropDataLink
}

var _ StreamDropCataloger = &StreamDropCatalog{}
type StreamDropCatalog struct {}

func (*StreamDropCatalog) GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) StreamDropDataLink {
	switch sdl := ctx.GetPublishedLink(StreamDropKey(pn)); {
	case sdl.IsZero():
		break
	case sdl.IsAssignableTo(&StreamDropDataLink{}):
		return StreamDropDataLink{sdl}
	default:
		return StreamDropDataLink{}
	}

	ctx.InitChild(StreamDropCreate())

	switch sdl := ctx.GetPublishedLink(StreamDropKey(pn)); {
	case sdl.IsZero():
		panic(throw.IllegalState())
	case sdl.IsAssignableTo(&StreamDropDataLink{}):
		return StreamDropDataLink{sdl}
	default:
		panic(throw.IllegalState())
	}
}

func StreamDropCreate() smachine.CreateFunc {
	return func(smachine.ConstructionContext) smachine.StateMachine {
		return &SMStreamDropBuilder{}
	}
}

type StreamDropDataLink struct {
	smachine.SharedDataLink
}

func (v StreamDropDataLink) GetSharedData(ctx smachine.ExecutionContext) (sd *StreamSharedData) {
	if !v.SharedDataLink.IsUnbound() {
		panic(throw.IllegalState())
	}

	if v.SharedDataLink.PrepareAccess(func(i interface{}) (wakeup bool) {
		sd = i.(*StreamSharedData)
		return false
	}).TryUse(ctx).GetDecision() != smachine.Passed {
		panic(throw.Impossible())
	}

	return
}
