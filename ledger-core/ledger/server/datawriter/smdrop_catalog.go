// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type JetDropKey jet.DropID

type DropCataloger interface {
	Create(ctx smachine.ExecutionContext, dropCfg DropConfig) DropDataLink
	Get(ctx smachine.SharedStateContext, dropID jet.DropID) DropDataLink
}

type JetTreeOp uint8

const (
	JetStraight JetTreeOp = iota
	JetGenesisSplit
	JetSplit
	JetMerge
)

type DropConfig struct {
	ID jet.DropID
	PrevID jet.DropID
	LastOp JetTreeOp
	// LegID jet.LegID // TODO LegID
}

var _ DropCataloger = DropCatalog{}
type DropCatalog struct {}

func (DropCatalog) Create(ctx smachine.ExecutionContext, dropCfg DropConfig) DropDataLink {

	if ctx.GetPublished(JetDropKey(dropCfg.ID)) != nil {
		panic(throw.IllegalState())
	}

	ctx.InitChild(JetDropCreate(dropCfg))

	switch sdl := ctx.GetPublishedLink(JetDropKey(dropCfg.ID)); {
	case sdl.IsZero():
		panic(throw.IllegalState())
	case sdl.IsAssignableTo(&DropSharedData{}):
		return DropDataLink{sdl}
	default:
		panic(throw.IllegalState())
	}
}

func (DropCatalog) Get(ctx smachine.SharedStateContext, dropID jet.DropID) DropDataLink {
	switch sdl := ctx.GetPublishedLink(JetDropKey(dropID)); {
	case sdl.IsZero():
	case sdl.IsAssignableTo(&DropSharedData{}):
		return DropDataLink{sdl}
	}
	return DropDataLink{}
}

func JetDropCreate(dropCfg DropConfig) smachine.CreateFunc {
	return func(smachine.ConstructionContext) smachine.StateMachine {
		sm := &SMDropBuilder{}
		sm.sd.id = dropCfg.ID
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
}

type DropDataLink struct {
	smachine.SharedDataLink
}

func (v DropDataLink) TryAccess(fn func(sd *DropSharedData)) smachine.BoolDecision {
	if fn == nil {
		panic(throw.IllegalValue())
	}

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
}
