// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PlashKey pulse.Number

type PlashCataloger interface {
	GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) *PlashSharedData
	Get(ctx smachine.SharedStateContext, pn pulse.Number) *PlashSharedData
}

var _ PlashCataloger = PlashCatalog{}
type PlashCatalog struct {}

func (PlashCatalog) GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) *PlashSharedData {
	sdl := ctx.GetPublishedLink(PlashKey(pn))
	if !sdl.IsZero() {
		return sdl.TryDirectAccess().(*PlashSharedData)
	}

	ctx.InitChild(PlashCreate())

	sdl = ctx.GetPublishedLink(PlashKey(pn))
	if sdl.IsZero() {
		panic(throw.IllegalState())
	}
	return sdl.TryDirectAccess().(*PlashSharedData)
}

func (PlashCatalog) Get(ctx smachine.SharedStateContext, pn pulse.Number) *PlashSharedData {
	sdl := ctx.GetPublishedLink(PlashKey(pn))
	if !sdl.IsZero() {
		return sdl.TryDirectAccess().(*PlashSharedData)
	}
	panic(throw.IllegalState())
}

func PlashCreate() smachine.CreateFunc {
	return func(smachine.ConstructionContext) smachine.StateMachine {
		return &SMPlash{}
	}
}

func RegisterPlash(ctx smachine.SharedStateContext, pr pulse.Range) *PlashSharedData {
	if pr == nil {
		panic(throw.IllegalState())
	}

	sd := &PlashSharedData{
		ready: smsync.NewConditionalBool(false, fmt.Sprintf("Plash{%d}.ready", ctx.SlotLink().SlotID())),
		pr: pr,
	}

	sdl := ctx.Share(sd, smachine.ShareDataUnbound)
	if !ctx.Publish(PlashKey(pr.RightBoundData().PulseNumber), sdl) {
		ctx.Unshare(sdl)
		return nil
	}
	return sd
}

