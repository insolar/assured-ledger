// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type LineCataloger interface {
	GetOrCreate(ctx smachine.ExecutionContext, lineRef reference.Global) LineDataLink
}

var _ LineCataloger = LineCatalog{}
type LineCatalog struct {}

func (LineCatalog) GetOrCreate(ctx smachine.ExecutionContext, lineRef reference.Global) LineDataLink {
	switch sdl := ctx.GetPublishedLink(LineKey(lineRef)); {
	case sdl.IsZero():
		break
	case sdl.IsAssignableTo(&LineSharedData{}):
		return LineDataLink{sdl}
	default:
		return LineDataLink{}
	}

	ctx.InitChild(LineCreate(lineRef))

	switch sdl := ctx.GetPublishedLink(LineKey(lineRef)); {
	case sdl.IsZero():
		panic(throw.IllegalState())
	case sdl.IsAssignableTo(&LineSharedData{}):
		return LineDataLink{sdl}
	default:
		panic(throw.IllegalState())
	}
}

func LineCreate(lineRef reference.Global) smachine.CreateFunc {
	return func(ctx smachine.ConstructionContext) smachine.StateMachine {
		sm := &SMLine{}
		sm.sd.lineRef = lineRef
		return sm
	}
}

func RegisterLine(ctx smachine.SharedStateContext, sd *LineSharedData) bool {
	switch {
	case sd.lineRef.IsZero():
		panic(throw.IllegalState())
	case !sd.limiter.IsZero():
		panic(throw.IllegalState())
	}

	sd.limiter = smsync.NewSemaphore(0, fmt.Sprintf("SMLine{%d}.limiter", ctx.SlotLink().SlotID()))

	sdl := ctx.Share(sd, 0)
	if !ctx.Publish(LineKey(sd.lineRef), sdl) {
		ctx.Unshare(sdl)
		return false
	}
	return true
}

type LineDataLink struct {
	smachine.SharedDataLink
}

func (v LineDataLink) PrepareAccess(fn func(sd *LineSharedData) (wakeup bool)) smachine.SharedDataAccessor {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return v.SharedDataLink.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*LineSharedData))
	})
}

func (v LineDataLink) TryAccess(ctx smachine.ExecutionContext, fn func(sd *LineSharedData) (wakeup bool)) smachine.Decision {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return v.SharedDataLink.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*LineSharedData))
	}).TryUse(ctx).GetDecision()
}
