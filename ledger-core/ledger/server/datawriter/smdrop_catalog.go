// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type JetDropID uint64

func NewJetDropID(pn pulse.Number, id buildersvc.JetID) JetDropID {

}


type JetDropKey JetDropID

type DropCataloger interface {
	GetOrCreate(ctx smachine.ExecutionContext, dropID JetDropID) DropDataLink
}

var _ DropCataloger = &DropCatalog{}
type DropCatalog struct {}

func (*DropCatalog) GetOrCreate(ctx smachine.ExecutionContext, dropID JetDropID) DropDataLink {
	switch sdl := ctx.GetPublishedLink(JetDropKey(dropID)); {
	case sdl.IsZero():
		break
	case sdl.IsAssignableTo(&DropDataLink{}):
		return DropDataLink{sdl}
	default:
		return DropDataLink{}
	}

	ctx.InitChild(DropCreate(dropID))

	switch sdl := ctx.GetPublishedLink(JetDropKey(dropID)); {
	case sdl.IsZero():
		panic(throw.IllegalState())
	case sdl.IsAssignableTo(&DropDataLink{}):
		return DropDataLink{sdl}
	default:
		panic(throw.IllegalState())
	}
}

func DropCreate(dropID JetDropID) smachine.CreateFunc {

}

type DropDataLink struct {
	smachine.SharedDataLink
}

func (v DropDataLink) PrepareAccess(fn func(sd *DropSharedData) (wakeup bool)) smachine.SharedDataAccessor {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return v.SharedDataLink.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*DropSharedData))
	})
}

func (v DropDataLink) TryAccess(ctx smachine.ExecutionContext, fn func(sd *DropSharedData) (wakeup bool)) smachine.Decision {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return v.SharedDataLink.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*DropSharedData))
	}).TryUse(ctx).GetDecision()
}
