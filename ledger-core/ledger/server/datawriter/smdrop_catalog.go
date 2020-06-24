// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
<<<<<<< HEAD
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type JetDropKey jet.DropID

type DropCataloger interface {
	Create(ctx smachine.ExecutionContext, dropID jet.LegID, updater buildersvc.JetDropAssistant) DropDataLink
	Get(ctx smachine.SharedStateContext, dropID jet.DropID) DropDataLink
=======
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
>>>>>>> Ledger SMs
}

var _ DropCataloger = &DropCatalog{}
type DropCatalog struct {}

<<<<<<< HEAD
func (*DropCatalog) Create(ctx smachine.ExecutionContext, legID jet.LegID, updater buildersvc.JetDropAssistant) DropDataLink {
	dropID := legID.AsDrop()
	if ctx.GetPublished(JetDropKey(dropID)) != nil {
		panic(throw.IllegalState())
	}

	ctx.InitChild(JetDropCreate(legID, updater))

	switch sdl := ctx.GetPublishedLink(JetDropKey(dropID)); {
	case sdl.IsZero():
		panic(throw.IllegalState())
	case sdl.IsAssignableTo(&DropDataLink{}):
		return DropDataLink{sdl}
	default:
		panic(throw.IllegalState())
	}
}

func (*DropCatalog) Get(ctx smachine.SharedStateContext, dropID jet.DropID) DropDataLink {
	switch sdl := ctx.GetPublishedLink(JetDropKey(dropID)); {
	case sdl.IsZero():
	case sdl.IsAssignableTo(&DropDataLink{}):
		return DropDataLink{sdl}
	}
	return DropDataLink{}
}

func JetDropCreate(legID jet.LegID, updater buildersvc.JetDropAssistant) smachine.CreateFunc {
	return func(smachine.ConstructionContext) smachine.StateMachine {
		sm := &SMJetDropBuilder{}
		sm.sd.id = legID
		sm.sd.updater = updater
		return sm
	}
}

func RegisterJetDrop(ctx smachine.SharedStateContext, sd *DropSharedData) bool {
	switch {
	case !sd.id.IsValid():
		panic(throw.IllegalState())
	case !sd.ready.IsZero():
		panic(throw.IllegalState())
	}

	sd.ready = smsync.NewConditionalBool(false, fmt.Sprintf("StreamDrop{%d}.ready", ctx.SlotLink().SlotID()))

	sdl := ctx.Share(sd, smachine.ShareDataDirect)
	if !ctx.Publish(JetDropKey(sd.id), sdl) {
		ctx.Unshare(sdl)
		return false
	}
	return true
=======
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

>>>>>>> Ledger SMs
}

type DropDataLink struct {
	smachine.SharedDataLink
}

<<<<<<< HEAD
func (v DropDataLink) TryAccess(fn func(sd *DropSharedData)) smachine.BoolDecision {
=======
func (v DropDataLink) PrepareAccess(fn func(sd *DropSharedData) (wakeup bool)) smachine.SharedDataAccessor {
>>>>>>> Ledger SMs
	if fn == nil {
		panic(throw.IllegalValue())
	}

<<<<<<< HEAD
	if i := v.SharedDataLink.TryDirectAccess(); i != nil {
		fn(i.(*DropSharedData))
		return true
	}
	return false
}

func (v DropDataLink) MustAccess(fn func(sd *DropSharedData)) {
	if !v.TryAccess(fn) {
		panic(throw.IllegalState())
	}
=======
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
>>>>>>> Ledger SMs
}
