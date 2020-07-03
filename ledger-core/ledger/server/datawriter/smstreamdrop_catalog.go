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

type StreamDropKey pulse.Number

type StreamDropCataloger interface {
	GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) *StreamSharedData
}

var _ StreamDropCataloger = &StreamDropCatalog{}
type StreamDropCatalog struct {}

func (*StreamDropCatalog) GetOrCreate(ctx smachine.ExecutionContext, pn pulse.Number) *StreamSharedData {
	sdl := ctx.GetPublishedLink(StreamDropKey(pn))
	if !sdl.IsZero() {
		return sdl.TryDirectAccess().(*StreamSharedData)
	}

	ctx.InitChild(StreamDropCreate())

	sdl = ctx.GetPublishedLink(StreamDropKey(pn))
	if sdl.IsZero() {
		panic(throw.IllegalState())
	}
	return sdl.TryDirectAccess().(*StreamSharedData)
}

func StreamDropCreate() smachine.CreateFunc {
	return func(smachine.ConstructionContext) smachine.StateMachine {
		return &SMStreamDropBuilder{}
	}
}

func RegisterStreamDrop(ctx smachine.SharedStateContext, pr pulse.Range) *StreamSharedData {
	if pr == nil {
		panic(throw.IllegalState())
	}

	sd := &StreamSharedData{
		ready: smsync.NewConditionalBool(false, fmt.Sprintf("StreamDrop{%d}.ready", ctx.SlotLink().SlotID())),
		pr: pr,
	}

	sdl := ctx.Share(sd, smachine.ShareDataUnbound)
	if !ctx.Publish(StreamDropKey(pr.RightBoundData().PulseNumber), sdl) {
		ctx.Unshare(sdl)
		return nil
	}
	return sd
}

